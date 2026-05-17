package conversation

import (
	"fmt"

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

func (a *ManagerActor) handleCreateGroup(ctx actor.Context, msg CreateGroupConvCmd) {
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
		MemberLimit: conv.MemberLimit,
		CreatedAt:   conv.CreatedAt,
	}})
}

func (a *ManagerActor) spawnConvActor(convID uint64) error {
	name := fmt.Sprintf("conv:%d", convID)
	_, err := a.engine.GetOrCreate(name, func() actor.Actor {
		return NewConversationActor(convID, a.store, a.engine, a.events)
	})
	return err
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
