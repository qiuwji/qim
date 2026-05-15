package message

import "qim/internal/actor"

type MessageStoreActor struct {
	engine *actor.Engine
}

func NewMessageStoreActor(engine *actor.Engine) *MessageStoreActor {
	return &MessageStoreActor{engine: engine}
}

func (a *MessageStoreActor) Receive(ctx actor.Context) {}
