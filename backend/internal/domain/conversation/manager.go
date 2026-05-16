package conversation

import (
	"fmt"

	"qim/internal/actor"
	"qim/internal/eventbus"
)

type ManagerActor struct {
	store  Store
	engine *actor.Engine
	events eventbus.Bus
}

func NewManagerActor(store Store, engine *actor.Engine, events eventbus.Bus) *ManagerActor {
	return &ManagerActor{store: store, engine: engine, events: events}
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
		result = append(result, UserConvDTO{
			ConversationID: uc.ConversationID,
			IsPinned:       uc.IsPinned,
			IsMuted:        uc.IsMuted,
			UnreadCount:    uc.UnreadCount,
			LastMsgAt:      uc.LastMsgAt,
		})
	}
	ctx.Reply(Result{Data: result})
}

func (a *ManagerActor) handleCreatePrivate(ctx actor.Context, msg CreatePrivateConvCmd) {
	conv, err := a.store.CreatePrivateConversation(CreatePrivateConversationInput{
		UID1: msg.UID1,
		UID2: msg.UID2,
	})
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	a.spawnConvActor(conv.ID)

	ctx.Reply(Result{Data: ConversationDTO{
		ID:      conv.ID,
		Type:    ConvType(conv.Type),
		OwnerID: conv.OwnerID,
	}})
}

func (a *ManagerActor) handleCreateGroup(ctx actor.Context, msg CreateGroupConvCmd) {
	conv, err := a.store.CreateGroupConversation(CreateGroupConversationInput{
		OwnerID:    msg.OwnerID,
		Name:       msg.Name,
		Avatar:     msg.Avatar,
		MemberUIDs: msg.Members,
	})
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	a.spawnConvActor(conv.ID)

	ctx.Reply(Result{Data: ConversationDTO{
		ID:          conv.ID,
		Type:        ConvType(conv.Type),
		Name:        conv.Name,
		Avatar:      conv.Avatar,
		OwnerID:     conv.OwnerID,
		MemberLimit: conv.MemberLimit,
		CreatedAt:   conv.CreatedAt,
	}})
}

func (a *ManagerActor) spawnConvActor(convID uint64) {
	name := fmt.Sprintf("conv:%d", convID)
	a.engine.GetOrCreate(name, func() actor.Actor {
		return NewConversationActor(convID, a.store, a.engine, a.events)
	})
}

func (a *ManagerActor) handleReadAll(ctx actor.Context, msg ReadAllConvCmd) {
	ucs, err := a.store.GetUserConversations(msg.UID)
	if err != nil {
		ctx.Reply(Result{Err: err})
		return
	}
	for _, uc := range ucs {
		if uc.UnreadCount > 0 {
			_ = a.store.UpdateUserConversation(msg.UID, uc.ConversationID, map[string]any{"unread_count": 0})
		}
	}
	ctx.Reply(Result{Data: true})
}
