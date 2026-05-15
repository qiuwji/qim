package conversation

import "errors"

type ErrorCode string

const (
	ErrCodeEmptyMessage       ErrorCode = "conversation.empty_message"
	ErrCodeNotMember          ErrorCode = "conversation.not_member"
	ErrCodeMemberExists       ErrorCode = "conversation.member_exists"
	ErrCodeMemberLimitReached ErrorCode = "conversation.member_limit_reached"
	ErrCodeMemberNotFound     ErrorCode = "conversation.member_not_found"
	ErrCodeOwnerRequired      ErrorCode = "conversation.owner_required"
)

type DomainError struct {
	Code    ErrorCode
	Message string
}

func (e DomainError) Error() string {
	return e.Message
}

func newDomainError(code ErrorCode, message string) error {
	return DomainError{Code: code, Message: message}
}

type ErrorPayload struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func ToErrorPayload(err error) (ErrorPayload, bool) {
	var domainErr DomainError
	if !errors.As(err, &domainErr) {
		return ErrorPayload{}, false
	}
	return ErrorPayload{Code: domainErr.Code, Message: domainErr.Message}, true
}

var (
	ErrEmptyMessage       = newDomainError(ErrCodeEmptyMessage, "message content is empty")
	ErrNotMember          = newDomainError(ErrCodeNotMember, "not a member")
	ErrMemberExists       = newDomainError(ErrCodeMemberExists, "member already exists")
	ErrMemberLimitReached = newDomainError(ErrCodeMemberLimitReached, "member limit reached")
	ErrMemberNotFound     = newDomainError(ErrCodeMemberNotFound, "member not found")
	ErrOwnerRequired      = newDomainError(ErrCodeOwnerRequired, "only owner can perform this operation")
)
