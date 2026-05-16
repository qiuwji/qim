package friend

import "qim/internal/pkg/apperr"

type ErrorCode = apperr.Code

const (
	ErrCodeNotYourRequest ErrorCode = "friend.not_your_request"
	ErrCodeCannotAddSelf  ErrorCode = "friend.cannot_add_self"
)

var (
	ErrNotYourRequest = apperr.New(ErrCodeNotYourRequest, "not your request")
	ErrCannotAddSelf  = apperr.New(ErrCodeCannotAddSelf, "cannot add yourself")
)
