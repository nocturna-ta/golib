package router

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/response"
	"github.com/nocturna-ta/golib/response/rest"

	"net/http"
	"runtime/debug"
)

// GlobalErrorHandler serve as a default global error handler
func GlobalErrorHandler(ctx *fiber.Ctx, err error) error {
	log.WithFields(log.Fields{
		"error": err,
		"path":  string(ctx.Request().URI().Path()),
	}).ErrorWithCtx(ctx.Context(), "[router.GlobalErrorHandler] Processing error")

	resp := rest.NewJSONResponse().SetError(err)
	return resp.Send(ctx)
}

// panicHandlerFiber handle when panic happened within code
func panicHandlerFiber(c *fiber.Ctx, h fiber.Handler) (err error) {
	defer func() {
		// catch panic
		if r := recover(); r != nil {
			stackTrace := string(debug.Stack())

			log.WithFields(log.Fields{
				"path":        c.Request().URI().Path(),
				"stack-trace": stackTrace,
				"error":       fmt.Sprintf("%+v", err),
			}).ErrorWithCtx(c.Context(), "[router.panicHandler] panic have occurred")

			var ok bool
			if err, ok = r.(error); !ok {
				// set error so it will call global error handler
				err = &custerr.ErrChain{
					Message: "internal server error",
					Cause:   err,
					Code:    http.StatusInternalServerError,
					Type:    response.ErrInternalServerError,
				}
			}
		}
	}()

	err = h(c)
	return err
}

func panicHandler[T rest.Response](handler Handler[T]) Handler[T] {
	return func(ctx context.Context, req *Request) (resp *T, err error) {
		defer func() {
			// catch panic
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())

				log.WithFields(log.Fields{
					"path":        req.RawRequest().URI().Path(),
					"stack-trace": stackTrace,
					"error":       fmt.Sprintf("%+v", err),
				}).ErrorWithCtx(ctx, "[router.panicHandler] panic have occurred")

				var ok bool
				if err, ok = r.(error); !ok {
					// set error so it will call global error handler
					err = &custerr.ErrChain{
						Message: "internal server error",
						Cause:   err,
						Code:    http.StatusInternalServerError,
						Type:    response.ErrInternalServerError,
					}
				}
			}
		}()

		resp, err = handler(ctx, req)
		return resp, err
	}
}
