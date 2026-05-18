package conversation

import (
	"errors"

	"qim/internal/pkg/apperr"
)

type ErrorCode = apperr.Code

const (
	ErrCodeEmptyMessage         ErrorCode = "conversation.empty_message"
	ErrCodeNotMember            ErrorCode = "conversation.not_member"
	ErrCodeMemberExists         ErrorCode = "conversation.member_exists"
	ErrCodeMemberLimitReached   ErrorCode = "conversation.member_limit_reached"
	ErrCodeMemberNotFound       ErrorCode = "conversation.member_not_found"
	ErrCodeOwnerRequired        ErrorCode = "conversation.owner_required"
	ErrCodeAdminRequired        ErrorCode = "conversation.admin_required"
	ErrCodeInvalidRole          ErrorCode = "conversation.invalid_role"
	ErrCodeGroupRequired        ErrorCode = "conversation.group_required"
	ErrCodeMentionLimitExceeded ErrorCode = "conversation.mention_limit_exceeded"
	ErrCodeMentionAllForbidden  ErrorCode = "conversation.mention_all_forbidden"
)

type DomainError = apperr.Error

func newDomainError(code ErrorCode, message string) error {
	return apperr.New(code, message)
}

type ErrorPayload = apperr.Payload

func ToErrorPayload(err error) (ErrorPayload, bool) {
	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		return ErrorPayload{}, false
	}
	return apperr.ToPayload(err), true
}

var (
	ErrEmptyMessage         = newDomainError(ErrCodeEmptyMessage, "message content is empty")
	ErrNotMember            = newDomainError(ErrCodeNotMember, "not a member")
	ErrMemberExists         = newDomainError(ErrCodeMemberExists, "member already exists")
	ErrMemberLimitReached   = newDomainError(ErrCodeMemberLimitReached, "member limit reached")
	ErrMemberNotFound       = newDomainError(ErrCodeMemberNotFound, "member not found")
	ErrOwnerRequired        = newDomainError(ErrCodeOwnerRequired, "only owner can perform this operation")
	ErrAdminRequired        = newDomainError(ErrCodeAdminRequired, "only owner or admin can perform this operation")
	ErrInvalidRole          = newDomainError(ErrCodeInvalidRole, "invalid member role")
	ErrGroupRequired        = newDomainError(ErrCodeGroupRequired, "only group conversation can perform this operation")
	ErrMentionLimitExceeded = newDomainError(ErrCodeMentionLimitExceeded, "mention limit exceeded (max 50)")
	ErrMentionAllForbidden  = newDomainError(ErrCodeMentionAllForbidden, "only owner or admin can mention all")
)
