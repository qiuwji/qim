package call

type CallStatus int8

const (
	CallStatusMissed    CallStatus = 1
	CallStatusCompleted CallStatus = 2
)

type Call struct {
	ID        uint64
	CallerUID uint64
	CalleeUID uint64
	CallType  int8
	Status    CallStatus
	StartedAt int64
	EndedAt   int64
	Duration  int64
	EndReason string
	CreatedAt int64
}

type CallStore interface {
	Create(call *Call) error
	GetByID(id uint64) (*Call, error)
	ListByUser(uid uint64, offset, limit int) ([]Call, error)
}
