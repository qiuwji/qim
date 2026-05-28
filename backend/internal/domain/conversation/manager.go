package conversation

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation/store"
	"qim/internal/eventbus"
)

type ManagerActor struct {
	store  store.Store
	engine *actor.Engine
	events eventbus.Bus
}

func NewManagerActor(s store.Store, engine *actor.Engine, events eventbus.Bus) *ManagerActor {
	return &ManagerActor{store: s, engine: engine, events: events}
}

func (a *ManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ListUserConversationsCmd:
		a.handleListUserConversations(ctx, msg)
	case CreatePrivateConvCmd:
		a.handleCreatePrivate(ctx, msg)
	case CreateGroupConvCmd:
		a.handleCreateGroup(ctx, msg)
	case CreateBotSessionCmd:
		a.handleCreateBotSession(ctx, msg)
	case ReadAllConvCmd:
		a.handleReadAll(ctx, msg)
	case PinConvCmd:
		a.handlePin(ctx, msg)
	case MuteConvCmd:
		a.handleMute(ctx, msg)
	case ReadConvCmd:
		a.handleRead(ctx, msg)
	}
}

func (a *ManagerActor) handleListUserConversations(ctx actor.Context, msg ListUserConversationsCmd) {
	ucs, err := a.store.GetUserConversations(msg.UID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	result := make([]UserConvDTO, 0, len(ucs))
	for _, uc := range ucs {
		conv, err := a.store.GetConversation(uc.ConversationID)
		if err != nil {
			continue
		}
		members, err := a.store.GetMembers(uc.ConversationID)
		if err != nil {
			ctx.Reply(Result{Err: err})
			return
		}
		result = append(result, UserConvDTO{
			ConversationID: uc.ConversationID,
			IsPinned:       uc.IsPinned,
			IsMuted:        uc.IsMuted,
			UnreadCount:    uc.UnreadCount,
			LastMsgAt:      uc.LastMsgAt,
			Conv: &ConversationDTO{
				ID:          conv.ID,
				Type:        store.ConvType(conv.Type),
				Name:        conv.Name,
				Avatar:      conv.Avatar,
				OwnerID:     conv.OwnerID,
				MemberCount: len(members),
				MemberLimit: conv.MemberLimit,
				MaxSeq:      conv.MaxSeq,
				CreatedAt:   conv.CreatedAt,
			},
		})
	}
	ctx.Reply(Result{Data: result})
}

func (a *ManagerActor) handleCreatePrivate(ctx actor.Context, msg CreatePrivateConvCmd) {
	conv, err := a.store.CreatePrivateConversation(store.CreatePrivateConversationInput{
		UID1: msg.UID1,
		UID2: msg.UID2,
	})
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.spawnConvActor(conv.ID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	ctx.Reply(Result{Data: ConversationDTO{
		ID:      conv.ID,
		Type:    store.ConvType(conv.Type),
		OwnerID: conv.OwnerID,
	}})
}

func (a *ManagerActor) handleCreateBotSession(ctx actor.Context, msg CreateBotSessionCmd) {
	now := time.Now().Unix()
	conv := &store.ConversationRecord{
		Type:        int8(store.ConvTypeBotSession),
		OwnerID:     msg.OwnerUID,
		MemberLimit: 2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := a.store.CreateConversation(conv); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	members := []store.MemberRecord{
		{ConversationID: conv.ID, UserID: msg.OwnerUID, Role: int8(store.MemberRoleOwner), JoinTime: now},
		{ConversationID: conv.ID, UserID: msg.BotUID, Role: int8(store.MemberRoleRegular), JoinTime: now},
	}
	if err := a.store.CreateMembers(members); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	if err := a.spawnConvActor(conv.ID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: ConversationDTO{
		ID:          conv.ID,
		Type:        store.ConvTypeBotSession,
		OwnerID:     conv.OwnerID,
		MemberCount: 2,
		MemberLimit: 2,
		CreatedAt:   conv.CreatedAt,
	}})
}

func (a *ManagerActor) handleCreateGroup(ctx actor.Context, msg CreateGroupConvCmd) {
	memberUIDs := groupConversationMemberUIDs(msg.OwnerID, msg.Members)
	conv, err := a.store.CreateGroupConversation(store.CreateGroupConversationInput{
		OwnerID:    msg.OwnerID,
		Name:       msg.Name,
		Avatar:     msg.Avatar,
		MemberUIDs: msg.Members,
	})
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	systemMsg, err := a.appendGroupCreatedSystemMessage(conv, memberUIDs)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	if err := a.spawnConvActor(conv.ID); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: ConversationDTO{
		ID:          conv.ID,
		Type:        store.ConvType(conv.Type),
		Name:        conv.Name,
		Avatar:      conv.Avatar,
		OwnerID:     conv.OwnerID,
		MemberCount: len(memberUIDs),
		MemberLimit: conv.MemberLimit,
		MaxSeq:      systemMsg.Seq,
		CreatedAt:   conv.CreatedAt,
	}})
	a.publishGroupCreated(conv, systemMsg, memberUIDs)
}

func (a *ManagerActor) spawnConvActor(convID uint64) error {
	name := fmt.Sprintf("conv:%d", convID)
	_, err := a.engine.GetOrCreate(name, func() actor.Actor {
		return NewConversationActor(convID, a.store, a.engine, a.events)
	})
	return err
}

func (a *ManagerActor) appendGroupCreatedSystemMessage(conv *store.ConversationRecord, memberUIDs []uint64) (*store.MessageCommitResult, error) {
	now := conv.CreatedAt
	if now == 0 {
		now = time.Now().Unix()
	}
	content := "群聊已创建"
	if conv.Name != "" {
		content = fmt.Sprintf("群聊「%s」已创建", conv.Name)
	}
	return a.store.CommitMessage(store.MessageCommitInput{
		Message: store.MessageAppendInput{
			ConversationID: conv.ID,
			Seq:            1,
			SenderID:       conv.OwnerID,
			MsgType:        MsgTypeSystem,
			Content:        content,
			CreatedAt:      now,
		},
		UnreadProjection: store.UnreadProjectionInput{
			ConversationID: conv.ID,
			SenderID:       conv.OwnerID,
			MemberUIDs:     memberUIDs,
			LastMsgAt:      now,
		},
	})
}

func (a *ManagerActor) publishGroupCreated(conv *store.ConversationRecord, msg *store.MessageCommitResult, memberUIDs []uint64) {
	if a.events == nil || msg == nil || msg.Duplicated {
		return
	}
	copiedMembers := append([]uint64(nil), memberUIDs...)
	_ = a.events.Publish(ConversationUpdatedEvent{
		ConversationID: conv.ID,
		DisplayName:    conv.Name,
		Avatar:         conv.Avatar,
		MemberUIDs:     copiedMembers,
	})
	_ = a.events.Publish(MessageSentEvent{
		MessageID:      msg.MessageID,
		ConversationID: conv.ID,
		ConvType:       conv.Type,
		Seq:            msg.Seq,
		SenderID:       msg.SenderID,
		MemberUIDs:     copiedMembers,
		MsgType:        msg.MsgType,
		Content:        msg.Content,
		CreatedAt:      msg.CreatedAt,
	})
}

func groupConversationMemberUIDs(ownerID uint64, members []uint64) []uint64 {
	result := make([]uint64, 0, len(members)+1)
	seen := map[uint64]struct{}{}
	for _, uid := range append([]uint64{ownerID}, members...) {
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

func (a *ManagerActor) handleReadAll(ctx actor.Context, msg ReadAllConvCmd) {
	ucs, err := a.store.GetUserConversations(msg.UID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	for _, uc := range ucs {
		if uc.UnreadCount > 0 {
			if err := a.store.UpdateUserConversation(msg.UID, uc.ConversationID, map[string]any{"unread_count": 0}); err != nil {
				ctx.Reply(Result{Err: err})
				return
			}
		}
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handlePin(ctx actor.Context, msg PinConvCmd) {
	if err := a.store.UpdateUserConversation(msg.UID, msg.ConversationID, map[string]any{"is_pinned": msg.Pinned}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handleMute(ctx actor.Context, msg MuteConvCmd) {
	if err := a.store.UpdateUserConversation(msg.UID, msg.ConversationID, map[string]any{"is_muted": msg.Muted}); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}

func (a *ManagerActor) handleRead(ctx actor.Context, msg ReadConvCmd) {
	if err := a.store.MarkConversationRead(msg.UID, msg.ConversationID, msg.Seq); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	ctx.Reply(Result{Data: true})
}
