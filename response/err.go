package response

import "errors"

var (
	ErrBadRequest          = errors.New("bad request")
	ErrForbiddenResource   = errors.New("forbidden resource")
	ErrNotFound            = errors.New("not found")
	ErrInternalServerError = errors.New("internal server error")
	ErrTimeoutError        = errors.New("request timeout")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrConflict            = errors.New("conflict")
	ErrRequestTooLarge     = errors.New("request entity too large")
)
