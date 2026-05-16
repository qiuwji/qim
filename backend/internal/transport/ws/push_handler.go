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
		h.handleEvent(ctx, msg.Event)
	}
}

func (h *MessagePushHandler) handleEvent(ctx actor.Context, event eventbus.Event) {
	if h.presence == nil {
		return
	}

	switch e := event.(type) {
	case presencedomain.UserOnlineEvent:
		h.handlePresenceChange(ctx, e.UID, "online")
	case presencedomain.UserOfflineEvent:
		h.handlePresenceChange(ctx, e.UID, "offline")
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

func (h *MessagePushHandler) handlePresenceChange(ctx actor.Context, uid uint64, action string) {
	if h.friendMgr == nil {
		return
	}
	raw, err := h.friendMgr.Ask(friend.ListFriendsCmd{UID: uid}, presenceAskTimeout)
	if err != nil {
		zap.L().Warn("query friends for presence push failed", zap.Uint64("uid", uid), zap.Error(err))
		return
	}
	result, ok := raw.(friend.Result)
	if !ok || result.Err != nil {
		return
	}
	dtos, ok := result.Data.([]friend.FriendDTO)
	if !ok {
		return
	}
	data := map[string]any{"uid": uid}
	for _, f := range dtos {
		h.pushToUser(f.FriendUID, "presence", action, data)
	}
}

func routePushEvent(event eventbus.Event) (pushType string, action string, recipients []uint64, data any, ok bool) {
	switch e := event.(type) {
	case conversation.MessageSentEvent:
		return "message", "new", e.MemberUIDs, e, true
	case conversation.MessageRevokedEvent:
		return "message", "revoked", e.MemberUIDs, e, true
	case conversation.TypingEvent:
		return "typing", "indicator", e.MemberUIDs, e, true
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
