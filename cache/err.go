package cache

import (
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/response"
)

var (
	ErrNotFound = &custerr.ErrChain{
		Message: "[cache] not found",
		Code:    404,
		Type:    response.ErrNotFound,
	}
)
