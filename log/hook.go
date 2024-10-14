package log

import (
	libCtx "github.com/nocturna-ta/golib/context"
	"github.com/rs/zerolog"
)

type TracingHook struct{}

func (h TracingHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	var requestId string

	ctx := e.GetCtx()
	reqCtx, _ := libCtx.GetRequestContext(ctx)
	e.Interface("request-context", reqCtx)

	if reqCtx != nil {
		requestId = reqCtx.RequestId
	} else {
		requestId = libCtx.ReadRequestId(ctx)
	}

	e.Str("request-id", requestId)
}
