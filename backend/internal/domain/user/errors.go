package user

import "qim/internal/pkg/apperr"

const (
	ErrCodeInvalidCredentials ErrorCode = "user.invalid_credentials"
	ErrCodeIncorrectPassword  ErrorCode = "user.incorrect_password"
)

type ErrorCode = apperr.Code

var (
	ErrInvalidCredentials = apperr.New(ErrCodeInvalidCredentials, "invalid credentials")
	ErrIncorrectPassword  = apperr.New(ErrCodeIncorrectPassword, "incorrect old password")
)
