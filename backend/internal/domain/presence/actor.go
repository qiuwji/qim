package presence

import (
	"qim/internal/actor"
	"qim/internal/eventbus"
)

type PresenceActor struct {
	engine     *actor.Engine
	events     eventbus.Bus
	online     map[uint64]struct{}
	gateways   map[uint64]map[string]*actor.ActorRef
	gatewayUID map[string]uint64
}

func NewPresenceActor(engine *actor.Engine, events eventbus.Bus) *PresenceActor {
	return &PresenceActor{
		engine:     engine,
		events:     events,
		online:     make(map[uint64]struct{}),
		gateways:   make(map[uint64]map[string]*actor.ActorRef),
		gatewayUID: make(map[string]uint64),
	}
}

type UserConnected struct {
	UID     uint64
	Gateway *actor.ActorRef
}

type UserDisconnected struct {
	UID     uint64
	Gateway *actor.ActorRef
}

type GetGatewaysQuery struct {
	UID uint64
}

type GatewaysResult struct {
	UID      uint64
	Gateways []*actor.ActorRef
}

type UserOnlineEvent struct {
	UID uint64 `json:"uid"`
}

type UserOfflineEvent struct {
	UID uint64 `json:"uid"`
}

func (UserOnlineEvent) Name() string  { return EventUserOnline }
func (UserOfflineEvent) Name() string { return EventUserOffline }

const (
	EventUserOnline  = "presence.user_online"
	EventUserOffline = "presence.user_offline"
)

func (a *PresenceActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case UserConnected:
		a.handleConnected(ctx, msg)
	case UserDisconnected:
		a.handleDisconnected(ctx, msg)
	case GetGatewaysQuery:
		a.handleGetGateways(ctx, msg)
	case actor.Terminated:
		a.handleTerminated(msg.Who)
	}
}

func (a *PresenceActor) handleConnected(ctx actor.Context, msg UserConnected) {
	if msg.UID == 0 || msg.Gateway == nil {
		return
	}
	name := msg.Gateway.Name()
	_, wasOnline := a.online[msg.UID]
	if a.gateways[msg.UID] == nil {
		a.gateways[msg.UID] = make(map[string]*actor.ActorRef)
	}
	a.gateways[msg.UID][name] = msg.Gateway
	a.gatewayUID[name] = msg.UID
	a.online[msg.UID] = struct{}{}
	ctx.Watch(msg.Gateway)
	if !wasOnline && a.events != nil {
		_ = a.events.Publish(UserOnlineEvent{UID: msg.UID})
	}
}

func (a *PresenceActor) handleDisconnected(ctx actor.Context, msg UserDisconnected) {
	if msg.Gateway == nil {
		return
	}
	a.removeGateway(msg.UID, msg.Gateway.Name())
	ctx.Unwatch(msg.Gateway)
}

func (a *PresenceActor) handleGetGateways(ctx actor.Context, msg GetGatewaysQuery) {
	refs := make([]*actor.ActorRef, 0, len(a.gateways[msg.UID]))
	for _, ref := range a.gateways[msg.UID] {
		refs = append(refs, ref)
	}
	ctx.Reply(GatewaysResult{UID: msg.UID, Gateways: refs})
}

func (a *PresenceActor) handleTerminated(ref *actor.ActorRef) {
	if ref == nil {
		return
	}
	uid, ok := a.gatewayUID[ref.Name()]
	if !ok {
		return
	}
	a.removeGateway(uid, ref.Name())
}

func (a *PresenceActor) removeGateway(uid uint64, gatewayName string) {
	if uid == 0 {
		uid = a.gatewayUID[gatewayName]
	}
	delete(a.gatewayUID, gatewayName)
	if uid == 0 {
		return
	}
	delete(a.gateways[uid], gatewayName)
	if len(a.gateways[uid]) == 0 {
		delete(a.gateways, uid)
		delete(a.online, uid)
		if a.events != nil {
			_ = a.events.Publish(UserOfflineEvent{UID: uid})
		}
	}
}
