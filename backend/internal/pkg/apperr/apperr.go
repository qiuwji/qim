package apperr

import "errors"

type Code string

const (
	CodeOK             Code = "ok"
	CodeBadRequest     Code = "bad_request"
	CodeInvalidRequest Code = "invalid_request"
	CodeUnauthorized   Code = "unauthorized"
	CodeRateLimit      Code = "rate_limit"
	CodeInternal       Code = "internal_error"
)

type Error struct {
	Code    Code
	Message string
	Cause   error
}

func New(code Code, message string) error {
	return &Error{Code: code, Message: message}
}

func Wrap(code Code, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type Payload struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func ToPayload(err error) Payload {
	if err == nil {
		return Payload{Code: CodeOK}
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return Payload{Code: appErr.Code, Message: appErr.Error()}
	}

	return Payload{Code: CodeInternal, Message: err.Error()}
}

func Is(err error) bool {
	var appErr *Error
	return errors.As(err, &appErr)
}
