package call

import "qim/internal/pkg/apperr"

type ErrorCode = apperr.Code

const (
	ErrCodeInvalidType      ErrorCode = "call.invalid_type"
	ErrCodeSelfBusy         ErrorCode = "call.self_busy"
	ErrCodeOffline          ErrorCode = "call.offline"
	ErrCodeBusy             ErrorCode = "call.busy"
	ErrCodeSelfCall         ErrorCode = "call.self_call"
	ErrCodeNotFound         ErrorCode = "call.not_found"
	ErrCodeAlreadyAnswered  ErrorCode = "call.already_answered"
	ErrCodeForbidden        ErrorCode = "call.forbidden"
	ErrCodeInvalidState     ErrorCode = "call.invalid_state"
)

func newError(code ErrorCode, message string) error {
	return apperr.New(code, message)
}

var (
	ErrInvalidType     = newError(ErrCodeInvalidType, "invalid call type")
	ErrSelfBusy        = newError(ErrCodeSelfBusy, "you are already in a call")
	ErrOffline         = newError(ErrCodeOffline, "user is offline")
	ErrBusy            = newError(ErrCodeBusy, "user is in another call")
	ErrSelfCall        = newError(ErrCodeSelfCall, "cannot call yourself")
	ErrNotFound        = newError(ErrCodeNotFound, "call not found")
	ErrAlreadyAnswered = newError(ErrCodeAlreadyAnswered, "call already answered")
	ErrForbidden       = newError(ErrCodeForbidden, "not a participant of this call")
	ErrInvalidState    = newError(ErrCodeInvalidState, "call is not in expected state")
)
