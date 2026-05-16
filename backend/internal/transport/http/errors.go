package http

import "qim/internal/pkg/apperr"

func badRequest(err error) error {
	if err == nil {
		return apperr.New(apperr.CodeBadRequest, "bad request")
	}
	return apperr.Wrap(apperr.CodeBadRequest, err.Error(), err)
}

func internalError(err error) error {
	if err == nil {
		return apperr.New(apperr.CodeInternal, "internal error")
	}
	return apperr.Wrap(apperr.CodeInternal, err.Error(), err)
}
