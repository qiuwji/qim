package ws

import (
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
	"qim/internal/domain/friend"
	presencedomain "qim/internal/domain/presence"
	"qim/internal/eventbus"

	"go.uber.org/zap"
)

const presenceAskTimeout = time.Second

type MessagePushHandler struct {
	presence  *actor.ActorRef
	friendMgr *actor.ActorRef
}

func NewMessagePushActor(presence *actor.ActorRef, friendMgr *actor.ActorRef) *MessagePushHandler {
	return &MessagePushHandler{presence: presence, friendMgr: friendMgr}
}

func (h *MessagePushHandler) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case eventbus.EventEnvelope:
		h.handleEvent(msg.Event)
	case conversation.TypingPushCmd:
		h.handleTypingPush(msg)
	}
}

func (h *MessagePushHandler) handleEvent(event eventbus.Event) {
	if h.presence == nil {
		return
	}

	switch e := event.(type) {
	case conversation.MessageSentEvent:
		for _, uid := range uniqueUIDs(e.MemberUIDs) {
			h.pushToUser(uid, "message", "new", e)
		}
		if len(e.MentionUIDs) > 0 || e.MentionAll {
			go h.pushMention(e)
		}
		return
	case presencedomain.UserOnlineEvent:
		h.handlePresenceChange(e.UID, "online")
	case presencedomain.UserOfflineEvent:
		h.handlePresenceChange(e.UID, "offline")
	default:
		pushType, action, recipients, data, ok := routePushEvent(event)
		if !ok {
			return
		}
		for _, uid := range uniqueUIDs(recipients) {
			h.pushToUser(uid, pushType, action, data)
		}
	}
}

func (h *MessagePushHandler) handlePresenceChange(uid uint64, action string) {
	if h.friendMgr == nil {
		zap.L().Warn("presence push skipped: friendMgr is nil", zap.Uint64("uid", uid), zap.String("action", action))
		return
	}
	raw, err := h.friendMgr.Ask(friend.ListFriendsCmd{UID: uid}, presenceAskTimeout)
	if err != nil {
		zap.L().Warn("query friends for presence push failed", zap.Uint64("uid", uid), zap.Error(err))
		return
	}
	result, ok := raw.(friend.Result)
	if !ok || result.Err != nil {
		zap.L().Warn("presence push: friend list result error", zap.Uint64("uid", uid), zap.Bool("ok", ok))
		return
	}
	dtos, ok := result.Data.([]friend.FriendDTO)
	if !ok {
		zap.L().Warn("presence push: friend list type assertion failed", zap.Uint64("uid", uid))
		return
	}
	zap.L().Info("presence change push", zap.Uint64("uid", uid), zap.String("action", action), zap.Int("friend_count", len(dtos)))
	data := map[string]any{"uid": uid}
	for _, f := range dtos {
		h.pushToUser(f.FriendUID, "presence", action, data)
	}
	if action == "online" {
		friendUIDs := make([]uint64, 0, len(dtos))
		for _, f := range dtos {
			friendUIDs = append(friendUIDs, f.FriendUID)
		}
		raw, err := h.presence.Ask(presencedomain.BatchOnlineQuery{UIDs: friendUIDs}, presenceAskTimeout)
		if err != nil {
			zap.L().Warn("presence online: batch query failed", zap.Uint64("uid", uid), zap.Error(err))
			return
		}
		batchResult, ok := raw.(presencedomain.BatchOnlineResult)
		if !ok {
			zap.L().Warn("presence online: batch result type assertion failed", zap.Uint64("uid", uid))
			return
		}
		onlineCount := 0
		for friendUID, online := range batchResult.OnlineMap {
			if online {
				onlineCount++
				h.pushToUser(uid, "presence", "online", map[string]any{"uid": friendUID})
			}
		}
		zap.L().Info("presence online: pushed online friends to user", zap.Uint64("uid", uid), zap.Int("online_friend_count", onlineCount))
	}
}

func routePushEvent(event eventbus.Event) (pushType string, action string, recipients []uint64, data any, ok bool) {
	switch e := event.(type) {
	case conversation.MessageSentEvent:
		return "message", "new", e.MemberUIDs, e, true
	case conversation.MessageRevokedEvent:
		return "message", "revoked", e.MemberUIDs, e, true
	case conversation.ConversationUpdatedEvent:
		return "conversation", "updated", e.MemberUIDs, e, true
	case conversation.MemberJoinedEvent:
		return "member", "joined", e.MemberUIDs, e, true
	case conversation.MemberLeftEvent:
		return "member", "left", e.MemberUIDs, e, true
	case conversation.MemberKickedEvent:
		return "member", "kicked", e.MemberUIDs, e, true
	case conversation.OwnerTransferredEvent:
		return "member", "owner_transferred", e.MemberUIDs, e, true
	case conversation.GroupDissolvedEvent:
		return "conversation", "group_dissolved", e.MemberUIDs, e, true
	case friend.FriendRequestCreatedEvent:
		return "friend", "request", []uint64{e.ToUID}, e, true
	case friend.FriendRequestHandledEvent:
		if e.Accepted {
			return "friend", "accepted", []uint64{e.FromUID}, e, true
		}
		return "friend", "rejected", []uint64{e.FromUID}, e, true
	default:
		return "", "", nil, nil, false
	}
}

func (h *MessagePushHandler) handleTypingPush(cmd conversation.TypingPushCmd) {
	if h.presence == nil || cmd.ToUID == 0 {
		return
	}
	h.pushToUser(cmd.ToUID, "typing", "indicator", map[string]any{
		"conversation_id": cmd.ConversationID,
		"user_id":         cmd.FromUID,
	})
}

func (h *MessagePushHandler) pushToUser(uid uint64, pushType, action string, data any) {
	raw, err := h.presence.Ask(presencedomain.GetGatewaysQuery{UID: uid}, presenceAskTimeout)
	if err != nil {
		zap.L().Warn("query presence gateways failed", zap.Uint64("uid", uid), zap.String("type", pushType), zap.String("action", action), zap.Error(err))
		return
	}
	result, ok := raw.(presencedomain.GatewaysResult)
	if !ok {
		zap.L().Warn("unexpected presence gateways result", zap.Uint64("uid", uid), zap.String("type", pushType), zap.String("action", action))
		return
	}
	for _, gateway := range result.Gateways {
		if err := gateway.Tell(PushCmd{Type: pushType, Action: action, Data: data}); err != nil {
			zap.L().Warn("push to gateway failed", zap.Uint64("uid", uid), zap.String("gateway", gateway.Name()), zap.String("type", pushType), zap.String("action", action), zap.Error(err))
		}
	}
}

func (h *MessagePushHandler) pushMention(e conversation.MessageSentEvent) {
	mentionUIDs := make([]uint64, len(e.MentionUIDs))
	copy(mentionUIDs, e.MentionUIDs)

	if e.MentionAll {
		mentionUIDs = append(mentionUIDs, e.MemberUIDs...)
		mentionUIDs = uniqueUIDs(mentionUIDs)
	}

	for _, uid := range mentionUIDs {
		if uid == e.SenderID {
			continue
		}
		h.pushToUser(uid, "message", "mention", map[string]any{
			"conversation_id": e.ConversationID,
			"message_id":      e.MessageID,
		})
	}
}

func uniqueUIDs(uids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(uids))
	result := make([]uint64, 0, len(uids))
	for _, uid := range uids {
		if uid == 0 {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		result = append(result, uid)
	}
	return result
}
