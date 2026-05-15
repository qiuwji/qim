package user

import "qim/internal/actor"

type SessionActor struct {
	uid      uint64
	engine   *actor.Engine
	gateways map[string]*actor.ActorRef
	convRefs map[uint64]*actor.ActorRef
}

func NewSessionActor(uid uint64, engine *actor.Engine) *SessionActor {
	return &SessionActor{
		uid:      uid,
		engine:   engine,
		gateways: make(map[string]*actor.ActorRef),
		convRefs: make(map[uint64]*actor.ActorRef),
	}
}

func (a *SessionActor) Receive(ctx actor.Context) {}
