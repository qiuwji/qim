package eventbus

import (
	"qim/internal/actor"

	"go.uber.org/zap"
)

type EventBusActor struct {
	subscribers map[string]map[string]*actor.ActorRef
}

func NewEventBusActor() *EventBusActor {
	return &EventBusActor{
		subscribers: make(map[string]map[string]*actor.ActorRef),
	}
}

func (a *EventBusActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case SubscribeCmd:
		a.subscribe(ctx, msg)
	case UnsubscribeCmd:
		a.unsubscribe(ctx, msg)
	case PublishCmd:
		a.publish(msg.Event)
	case actor.Terminated:
		a.removeSubscriber(msg.Who)
	}
}

func (a *EventBusActor) subscribe(ctx actor.Context, msg SubscribeCmd) {
	if msg.EventName == "" || msg.Subscriber == nil {
		return
	}
	if a.subscribers[msg.EventName] == nil {
		a.subscribers[msg.EventName] = make(map[string]*actor.ActorRef)
	}
	a.subscribers[msg.EventName][msg.Subscriber.Name()] = msg.Subscriber
	ctx.Watch(msg.Subscriber)
}

func (a *EventBusActor) unsubscribe(ctx actor.Context, msg UnsubscribeCmd) {
	if msg.EventName == "" || msg.Subscriber == nil {
		return
	}
	subs := a.subscribers[msg.EventName]
	if subs == nil {
		return
	}
	delete(subs, msg.Subscriber.Name())
	if len(subs) == 0 {
		delete(a.subscribers, msg.EventName)
	}
	if !a.hasSubscriber(msg.Subscriber.Name()) {
		ctx.Unwatch(msg.Subscriber)
	}
}

func (a *EventBusActor) publish(event Event) {
	if event == nil {
		return
	}
	for _, subscriber := range a.subscribers[event.Name()] {
		if err := subscriber.Tell(EventEnvelope{Event: event}); err != nil {
			zap.L().Warn("event delivery failed", zap.String("event", event.Name()), zap.String("subscriber", subscriber.Name()), zap.Error(err))
		}
	}
}

func (a *EventBusActor) removeSubscriber(ref *actor.ActorRef) {
	if ref == nil {
		return
	}
	name := ref.Name()
	for eventName, subs := range a.subscribers {
		delete(subs, name)
		if len(subs) == 0 {
			delete(a.subscribers, eventName)
		}
	}
}

func (a *EventBusActor) hasSubscriber(name string) bool {
	for _, subs := range a.subscribers {
		if _, ok := subs[name]; ok {
			return true
		}
	}
	return false
}

type RefBus struct {
	ref *actor.ActorRef
}

func NewRefBus(ref *actor.ActorRef) *RefBus {
	return &RefBus{ref: ref}
}

func (b *RefBus) Publish(event Event) error {
	if b == nil || b.ref == nil || event == nil {
		return nil
	}
	return b.ref.Tell(PublishCmd{Event: event})
}

func (b *RefBus) Subscribe(eventName string, subscriber *actor.ActorRef) error {
	if b == nil || b.ref == nil || eventName == "" || subscriber == nil {
		return nil
	}
	return b.ref.Tell(SubscribeCmd{EventName: eventName, Subscriber: subscriber})
}

func (b *RefBus) Unsubscribe(eventName string, subscriber *actor.ActorRef) error {
	if b == nil || b.ref == nil || eventName == "" || subscriber == nil {
		return nil
	}
	return b.ref.Tell(UnsubscribeCmd{EventName: eventName, Subscriber: subscriber})
}
