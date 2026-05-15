package presence

import "qim/internal/actor"

type PresenceActor struct {
	engine   *actor.Engine
	online   map[uint64]struct{}
	watchers map[uint64]map[uint64]struct{}
}

func NewPresenceActor(engine *actor.Engine) *PresenceActor {
	return &PresenceActor{
		engine:   engine,
		online:   make(map[uint64]struct{}),
		watchers: make(map[uint64]map[uint64]struct{}),
	}
}

func (a *PresenceActor) Receive(ctx actor.Context) {}
