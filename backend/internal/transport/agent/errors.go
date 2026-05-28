package agent

import "qim/internal/pkg/apperr"

var (
	ErrUnauthorized     = apperr.New(apperr.Code("agent.unauthorized"), "invalid platform token")
	ErrBotNotFound      = apperr.New(apperr.Code("agent.bot_not_found"), "bot user not found")
	ErrPermissionDenied = apperr.New(apperr.Code("agent.permission_denied"), "bot lacks required permission")
	ErrApprovalDenied   = apperr.New(apperr.Code("agent.approval_denied"), "human approval rejected or timeout")
	ErrInvalidEvent     = apperr.New(apperr.Code("agent.invalid_event"), "event type not in whitelist")
	ErrRateLimited      = apperr.New(apperr.Code("agent.rate_limited"), "too many requests")
)
