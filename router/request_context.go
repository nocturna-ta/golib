package router

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	libCtx "github.com/nocturna-ta/golib/context"
)

// requestContextHandler trying to get RequestContext from request header and save it in current context
// RequestContext value might be empty if there is no header found (can be from public call)
func requestContextHandler(c *fiber.Ctx) error {
	var userId, requestId, channelId string

	userId = string(c.Request().Header.Peek(libCtx.XUserId))

	requestId = string(c.Request().Header.Peek(libCtx.XRequestId))
	if requestId == "" {
		requestId = uuid.New().String()
	}

	channelId = string(c.Request().Header.Peek(libCtx.XChannelId))

	reqCtx := libCtx.RequestContext{
		UserId:    userId,
		RequestId: requestId,
		ChannelId: channelId,
	}

	ctx := c.UserContext()
	ctx = context.WithValue(ctx, libCtx.RequestIdKey, requestId)
	ctx = context.WithValue(ctx, libCtx.RequestContextKey, reqCtx)

	c.SetUserContext(ctx)

	return c.Next()
}

func validateRequestContext(ctx context.Context) error {
	rc, err := libCtx.GetRequestContext(ctx)
	if err != nil {
		return errUnauthorized
	}

	if rc.UserId == "" {
		return errUnauthorized
	}

	_, err = uuid.Parse(rc.UserId)
	if err != nil {
		return errUnauthorized
	}

	return nil
}
