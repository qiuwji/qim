package agent

import (
	"io"
	"net/http"

	"qim/internal/actor"
)

type Result struct {
	Data any
	Err  error
}

type SubscribeCmd struct {
	SessionID string
	BotUID    uint64
	Events    []string
	Filter    *EventFilter
}

type UnsubscribeCmd struct {
	SessionID string
	BotUID    uint64
	Events    []string
}

type EventFilter struct {
	ConvID *uint64
}

type PushNotificationCmd struct {
	SessionID string
	BotUID    uint64
	Type      string
	Event     any
}

type RefreshPermissionsCmd struct {
	BotUID uint64
}

type PermissionQuery struct {
	BotUID     uint64
	Permission string
}

type SetupSSECmd struct {
	SessionID string
	Writer    io.Writer
	Flusher   http.Flusher
}

type SSEConnected struct {
	SessionID string
}

type SSEHeartbeat struct {
	SessionID string
}

type SSEDisconnected struct {
	SessionID string
}

type SetSessionCmd struct {
	SessionID string
}

type RateLimitQuery struct {
	SessionID string
	BotUID    uint64
	ToolName  string
}

type AgentIdleTimeout struct {
	SessionID string
}

type AgentRefResolved struct {
	GwRef *actor.ActorRef
}
