package router

import (
	"golib/custerr"
	"golib/response"
	"net/http"
)

var (
	errUnauthorized = &custerr.ErrChain{
		Message: "unauthorized",
		Code:    http.StatusUnauthorized,
		Type:    response.ErrUnauthorized,
	}
)
