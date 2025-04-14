package router

import (
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/response"
	"net/http"
)

var (
	errUnauthorized = &custerr.ErrChain{
		Message: "unauthorized",
		Code:    http.StatusUnauthorized,
		Type:    response.ErrUnauthorized,
	}

	errForbiddenRole = &custerr.ErrChain{
		Message: "forbidden role",
		Code:    http.StatusForbidden,
		Type:    response.ErrForbiddenResource,
	}
)
