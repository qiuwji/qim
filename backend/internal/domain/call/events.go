package call

const (
	EventCallIncoming          = "call.incoming"
	EventCallCalling           = "call.calling"
	EventCallAccepted          = "call.accepted"
	EventCallRejected          = "call.rejected"
	EventCallCancelled         = "call.cancelled"
	EventCallEnded             = "call.ended"
	EventCallTimeout           = "call.timeout"
	EventCallAnsweredElsewhere = "call.answered_elsewhere"
)

type CallIncomingEvent struct {
	CallID       string `json:"call_id"`
	CallerUID    uint64 `json:"caller_uid"`
	CalleeUID    uint64 `json:"-"`
	CallType     int8   `json:"call_type"`
	CallerName   string `json:"caller_nickname"`
	CallerAvatar string `json:"caller_avatar"`
}

func (CallIncomingEvent) Name() string { return EventCallIncoming }

type CallCallingEvent struct {
	CallID    string `json:"call_id"`
	CallerUID uint64 `json:"-"`
	CalleeUID uint64 `json:"callee_uid"`
}

func (CallCallingEvent) Name() string { return EventCallCalling }

type CallAcceptedEvent struct {
	CallID    string `json:"call_id"`
	CallerUID uint64 `json:"-"`
	CalleeUID uint64 `json:"-"`
}

func (CallAcceptedEvent) Name() string { return EventCallAccepted }

type CallRejectedEvent struct {
	CallID    string `json:"call_id"`
	CallerUID uint64 `json:"-"`
	CalleeUID uint64 `json:"-"`
}

func (CallRejectedEvent) Name() string { return EventCallRejected }

type CallCancelledEvent struct {
	CallID    string `json:"call_id"`
	CallerUID uint64 `json:"-"`
	CalleeUID uint64 `json:"-"`
}

func (CallCancelledEvent) Name() string { return EventCallCancelled }

type CallEndedEvent struct {
	CallID    string `json:"call_id"`
	CallerUID uint64 `json:"-"`
	CalleeUID uint64 `json:"-"`
	StartedAt int64  `json:"started_at"`
	Duration  int64  `json:"duration"`
	EndReason string `json:"end_reason"`
}

func (CallEndedEvent) Name() string { return EventCallEnded }

type CallTimeoutEvent struct {
	CallID    string `json:"call_id"`
	CallerUID uint64 `json:"-"`
	CalleeUID uint64 `json:"-"`
}

func (CallTimeoutEvent) Name() string { return EventCallTimeout }

type CallAnsweredElsewhereEvent struct {
	CallID    string `json:"call_id"`
	CallerUID uint64 `json:"-"`
	CalleeUID uint64 `json:"-"`
	AcceptGW  string `json:"-"`
}

func (CallAnsweredElsewhereEvent) Name() string { return EventCallAnsweredElsewhere }
