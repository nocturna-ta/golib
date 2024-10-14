package cache

import (
	"golib/custerr"
	"golib/response"
)

var (
	ErrNotFound = &custerr.ErrChain{
		Message: "[cache] not found",
		Code:    404,
		Type:    response.ErrNotFound,
	}
)
