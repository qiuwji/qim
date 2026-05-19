package call

const (
	CallTypeVoice int8 = 1
	CallTypeVideo int8 = 2
)

const (
	CallStatusRinging   = "ringing"
	CallStatusConnected = "connected"
	CallStatusEnded     = "ended"
)

type Result struct {
	Data any
	Err  error
}

// --- 命令 ---

type InitiateCallCmd struct {
	CallerUID uint64
	CalleeUID uint64
	CallType  int8
}

type AcceptCallCmd struct {
	CallID      string
	UID         uint64
	GatewayName string
}

type RejectCallCmd struct {
	CallID string
	UID    uint64
}

type CancelCallCmd struct {
	CallID string
	UID    uint64
}

type EndCallCmd struct {
	CallID string
	UID    uint64
}

type ForwardOfferCmd struct {
	CallID string
	UID    uint64
	SDP    string
}

type ForwardAnswerCmd struct {
	CallID string
	UID    uint64
	SDP    string
}

type ForwardIceCmd struct {
	CallID        string
	UID           uint64
	Candidate     string
	SDPMid        string
	SDPMLineIndex int
}

// --- 内部消息 ---

type StartCallCmd struct {
	CallID    string
	CallerUID uint64
	CalleeUID uint64
	CallType  int8
	CallerGW  string
	CallerInfo CallerInfo
}

type CallerInfo struct {
	Nickname string
	Avatar   string
}

type TimeoutCallCmd struct{}

// --- 查询 ---

type GetCallByUserQuery struct {
	UID uint64
}

type InitiateResult struct {
	CallID string
}

type CallInfo struct {
	CallID    string
	CallerUID uint64
	CalleeUID uint64
	CallType  int8
	Status    string
}
