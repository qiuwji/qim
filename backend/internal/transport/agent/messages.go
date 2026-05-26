package agent

import "qim/internal/actor"

type Result struct {
	Data any
	Err  error
}

type SubscribeCmd struct {
	BotUID uint64
	Events []string
	Filter *EventFilter
}

type UnsubscribeCmd struct {
	BotUID uint64
	Events []string
}

type EventFilter struct {
	ConvID *uint64
}

type PushNotificationCmd struct {
	BotUID uint64
	Type   string
	Event  any
}

type RefreshPermissionsCmd struct {
	BotUID uint64
}

type SSEConnected struct{}

type SSEDisconnected struct{}

type AgentRefResolved struct {
	GwRef *actor.ActorRef
}
