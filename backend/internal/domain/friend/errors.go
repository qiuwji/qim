package friend

import "qim/internal/pkg/apperr"

type ErrorCode = apperr.Code

const (
	ErrCodeNotYourRequest   ErrorCode = "friend.not_your_request"
	ErrCodeCannotAddSelf    ErrorCode = "friend.cannot_add_self"
	ErrCodeAlreadyFriends   ErrorCode = "friend.already_friends"
	ErrCodePendingRequest   ErrorCode = "friend.pending_request"
)

var (
	ErrNotYourRequest = apperr.New(ErrCodeNotYourRequest, "not your request")
	ErrCannotAddSelf  = apperr.New(ErrCodeCannotAddSelf, "cannot add yourself")
	ErrAlreadyFriends = apperr.New(ErrCodeAlreadyFriends, "already friends")
	ErrPendingRequest = apperr.New(ErrCodePendingRequest, "a pending request already exists")
)
