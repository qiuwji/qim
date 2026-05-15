package conversation

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
)

type ManagerActor struct {
	store  dal.ConvStore
	engine *actor.Engine
}

func NewManagerActor(store dal.ConvStore, engine *actor.Engine) *ManagerActor {
	return &ManagerActor{store: store, engine: engine}
}

func (a *ManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case ListUserConversationsCmd:
		a.handleListUserConversations(ctx, msg)
	case CreatePrivateConvCmd:
		a.handleCreatePrivate(ctx, msg)
	case CreateGroupConvCmd:
		a.handleCreateGroup(ctx, msg)
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
	existing, err := a.store.FindPrivateConversation(msg.UID1, msg.UID2)
	if err == nil && existing != nil {
		ctx.Reply(Result{Data: ConversationDTO{
			ID:      existing.ID,
			Type:    ConvType(existing.Type),
			OwnerID: existing.OwnerID,
		}})
		return
	}

	now := time.Now().Unix()
	conv := &dal.Conversation{
		Type:        int8(ConvTypePrivate),
		OwnerID:     msg.UID1,
		MemberLimit: 2,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := a.store.CreateConversation(conv); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	members := []dal.Member{
		{ConversationID: conv.ID, UserID: msg.UID1, Role: int8(MemberRoleRegular), JoinTime: now},
		{ConversationID: conv.ID, UserID: msg.UID2, Role: int8(MemberRoleRegular), JoinTime: now},
	}
	a.store.CreateMembers(members)

	a.spawnConvActor(conv.ID)

	ctx.Reply(Result{Data: ConversationDTO{
		ID:      conv.ID,
		Type:    ConvType(conv.Type),
		OwnerID: conv.OwnerID,
	}})
}

func (a *ManagerActor) handleCreateGroup(ctx actor.Context, msg CreateGroupConvCmd) {
	now := time.Now().Unix()
	conv := &dal.Conversation{
		Type:        int8(ConvTypeGroup),
		Name:        msg.Name,
		Avatar:      msg.Avatar,
		OwnerID:     msg.OwnerID,
		MemberLimit: 500,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := a.store.CreateConversation(conv); err != nil {
		ctx.Reply(Result{Err: err})
		return
	}

	ownerMember := &dal.Member{
		ConversationID: conv.ID,
		UserID:         msg.OwnerID,
		Role:           int8(MemberRoleOwner),
		JoinTime:       now,
	}
	a.store.CreateMember(ownerMember)

	for _, uid := range msg.Members {
		if uid == msg.OwnerID {
			continue
		}
		m := &dal.Member{
			ConversationID: conv.ID,
			UserID:         uid,
			Role:           int8(MemberRoleRegular),
			JoinTime:       now,
		}
		a.store.CreateMember(m)
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
		return NewConversationActor(convID, a.store, a.engine)
	})
}
