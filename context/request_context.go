package context

import (
	"context"
	"github.com/google/uuid"
	"golib/custerr"
	"golib/response"
)

const (
	MetadataRetryAttempts = "md-retry-attempts"
	MetadataLogRefId      = "md-log-reference-id"
	XUserId               = "X-User-Id"
	XRequestId            = "X-Request-Id"
	XChannelId            = "X-Channel-Id"
	XAccountId            = "X-Account-Id"
)

var (
	RequestContextKey = requestContextKey{}
	RequestIdKey      = requestIdKey{}
)

type requestContextKey struct{}
type requestIdKey struct{}

type RequestContext struct {
	UserId    string `json:"user-id,omitempty"`
	RequestId string `json:"request-id,omitempty"`
	ChannelId string `json:"channel-id,omitempty"`
	AccountId string `json:"account-id,omitempty"`
}

func ReadRequestId(ctx context.Context) string {
	requestId, _ := ctx.Value(RequestIdKey).(string)
	if requestId == "" {
		reqCtx, _ := GetRequestContext(ctx)
		if reqCtx != nil {
			requestId = reqCtx.RequestId
		}
	}
	return requestId
}

func GetRequestContext(ctx context.Context) (*RequestContext, error) {
	reqCtx := ctx.Value(RequestContextKey)

	if reqCtx == nil {
		return nil, &custerr.ErrChain{
			Message: "Missing request context",
			Code:    400,
			Type:    response.ErrBadRequest,
		}
	}

	requestCtx, ok := reqCtx.(RequestContext)
	if !ok {
		return nil, &custerr.ErrChain{
			Message: "Missing request context",
			Code:    400,
			Type:    response.ErrBadRequest,
		}
	}

	return &requestCtx, nil
}

func (rc *RequestContext) GetAccountId() uuid.UUID {
	res, _ := uuid.Parse(rc.AccountId)
	return res
}

func (rc *RequestContext) GetUserId() uuid.UUID {
	res, _ := uuid.Parse(rc.UserId)
	return res
}
