package friend

import "qim/internal/pkg/apperr"

type ErrorCode = apperr.Code

const (
	ErrCodeNotYourRequest ErrorCode = "friend.not_your_request"
)

var (
	ErrNotYourRequest = apperr.New(ErrCodeNotYourRequest, "not your request")
)
