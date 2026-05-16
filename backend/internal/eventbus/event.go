package eventbus

import "qim/internal/actor"

type Event interface {
	Name() string
}

type EventEnvelope struct {
	Event Event
}

type PublishCmd struct {
	Event Event
}

type SubscribeCmd struct {
	EventName  string
	Subscriber *actor.ActorRef
}

type UnsubscribeCmd struct {
	EventName  string
	Subscriber *actor.ActorRef
}

type Bus interface {
	Publish(event Event) error
	Subscribe(eventName string, subscriber *actor.ActorRef) error
	Unsubscribe(eventName string, subscriber *actor.ActorRef) error
}
