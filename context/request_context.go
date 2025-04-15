package context

import (
	"context"
	"github.com/google/uuid"
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/response"
)

const (
	MetadataRetryAttempts = "md-retry-attempts"
	MetadataLogRefId      = "md-log-reference-id"
	XUserId               = "X-User-Id"
	XChannelId            = "X-Channel-Id"
	XRequestId            = "X-Request-Id"
	XAddressId            = "X-Address-Id"
	XRole                 = "X-Role"
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
	Address   string `json:"address-id,omitempty"`
	Role      string `json:"role,omitempty"`
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

func (rc *RequestContext) GetUserId() uuid.UUID {
	res, _ := uuid.Parse(rc.UserId)
	return res
}

func (rc *RequestContext) GetAddress() string {
	res := rc.Address
	return res
}

func (rc *RequestContext) GetRole() string {
	res := rc.Role
	return res
}

func (rc *RequestContext) HasRole(role string) bool {
	return rc.Role == role
}

func (rc *RequestContext) HasAnyRole(roles ...string) bool {
	if rc.Role == "" {
		return false
	}

	for _, r := range roles {
		if rc.Role == r {
			return true
		}
	}
	return false
}
