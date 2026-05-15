package conversation

import "qim/internal/actor"

type ConversationActor struct {
	convID      uint64
	convType    int8
	ownerID     uint64
	maxSeq      int64
	members     map[uint64]*MemberState
	memberLimit int
	storeRef    *actor.ActorRef
	presenceRef *actor.ActorRef
}

type MemberState struct {
	UID        uint64
	Role       int8
	LastReadSeq int64
	JoinTime   int64
	SessionRef *actor.ActorRef
}

func NewConversationActor(convID uint64) *ConversationActor {
	return &ConversationActor{
		convID:  convID,
		members: make(map[uint64]*MemberState),
	}
}

func (a *ConversationActor) Receive(ctx actor.Context) {}
