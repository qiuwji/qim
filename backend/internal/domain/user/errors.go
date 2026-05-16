package user

import "qim/internal/pkg/apperr"

const (
	ErrCodeInvalidCredentials ErrorCode = "user.invalid_credentials"
	ErrCodeIncorrectPassword  ErrorCode = "user.incorrect_password"
	ErrCodeInvalidUsername    ErrorCode = "user.invalid_username"
	ErrCodeWeakPassword       ErrorCode = "user.weak_password"
)

type ErrorCode = apperr.Code

var (
	ErrInvalidCredentials = apperr.New(ErrCodeInvalidCredentials, "invalid credentials")
	ErrIncorrectPassword  = apperr.New(ErrCodeIncorrectPassword, "incorrect old password")
	ErrInvalidUsername    = apperr.New(ErrCodeInvalidUsername, "username must contain digits only")
	ErrWeakPassword       = apperr.New(ErrCodeWeakPassword, "password must be 8-20 characters with uppercase, lowercase and special character")
)
