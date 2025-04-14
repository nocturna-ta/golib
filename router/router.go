package router

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	_ "github.com/newrelic/go-agent/v3/integrations/nrmysql"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/response"
	"github.com/nocturna-ta/golib/response/rest"
	"github.com/nocturna-ta/golib/tracing"

	newrelicLib "github.com/nocturna-ta/golib/tracing/newrelic"
	sentryLib "github.com/nocturna-ta/golib/tracing/sentry"
	"net/http"
	"strings"
	"time"
)

type (
	FastRouter struct {
		app      *fiber.App
		Options  *Options
		newRelic *newrelic.Application
	}

	Options struct {
		Prefix           string
		Port             uint
		ReadTimeout      time.Duration
		WriteTimeout     time.Duration
		RequestTimeout   time.Duration
		RequestBodyLimit int
		ErrorHandler     *fiber.ErrorHandler
		CorsConfig       *CorsConfig
		NewRelicOpts     *newrelicLib.Options
		SentryConfig     *sentryLib.Config
	}

	CorsConfig struct {
		AllowOrigins     string
		AllowMethods     string
		AllowHeaders     string
		AllowCredentials bool
		ExposeHeaders    string
		MaxAge           int
	}

	Handler[T rest.Response] func(ctx context.Context, req *Request) (*T, error)

	handlerResult[T rest.Response] struct {
		Resp *T
		Err  error
	}
)

func New(opt *Options) *FastRouter {
	errHandler := GlobalErrorHandler
	if opt.ErrorHandler != nil {
		errHandler = *opt.ErrorHandler
	}

	if opt.RequestTimeout == 0 {
		// if not set, then set default timeout
		opt.RequestTimeout = 10 * time.Second
	}

	if opt.Port == 0 {
		// set default port if not set
		opt.Port = 7000
	}

	config := fiber.Config{
		ReadTimeout:           opt.ReadTimeout,
		WriteTimeout:          opt.WriteTimeout,
		ErrorHandler:          errHandler,
		DisableStartupMessage: true,
		StreamRequestBody:     true,
	}

	if opt.RequestBodyLimit > 0 {
		config.BodyLimit = opt.RequestBodyLimit
	}

	app := fiber.New(config)

	if opt.CorsConfig != nil {
		app.Use(corsFromConfig(*opt.CorsConfig))
	}

	// register request context handler
	app.Use(requestContextHandler)

	if opt.SentryConfig != nil {
		if err := sentryLib.Init(opt.SentryConfig); err == nil {
			// register sentry
			app.Use(sentryLib.New(&sentryLib.Options{
				WaitForDelivery: false,
				Timeout:         3 * time.Second,
			}))
		}
	}

	router := &FastRouter{
		app:      app,
		Options:  opt,
		newRelic: newrelicLib.SetupNewRelic(opt.NewRelicOpts),
	}

	return router
}

func (jr *FastRouter) GET(path string, handler Handler[rest.JSONResponse], opts ...Option) {
	jr.Handle(http.MethodGet, path, handler, opts...)
}

func (jr *FastRouter) POST(path string, handler Handler[rest.JSONResponse], opts ...Option) {
	jr.Handle(http.MethodPost, path, handler, opts...)
}

func (jr *FastRouter) PUT(path string, handler Handler[rest.JSONResponse], opts ...Option) {
	jr.Handle(http.MethodPut, path, handler, opts...)
}

func (jr *FastRouter) PATCH(path string, handler Handler[rest.JSONResponse], opts ...Option) {
	jr.Handle(http.MethodPatch, path, handler, opts...)
}

func (jr *FastRouter) DELETE(path string, handler Handler[rest.JSONResponse], opts ...Option) {
	jr.Handle(http.MethodDelete, path, handler, opts...)
}

func (jr *FastRouter) ATTACHMENT(path string, handler Handler[rest.AttachmentResponse], opts ...Option) {
	jr.HandleAttachment(http.MethodGet, path, handler, opts...)
}

func (jr *FastRouter) Group(prefix string, fn func(r *FastRouter)) {
	nr := &FastRouter{
		app: jr.app,
		Options: &Options{
			Prefix:         jr.Options.Prefix + prefix,
			ReadTimeout:    jr.Options.ReadTimeout,
			WriteTimeout:   jr.Options.WriteTimeout,
			ErrorHandler:   jr.Options.ErrorHandler,
			RequestTimeout: jr.Options.RequestTimeout,
		},
		newRelic: jr.newRelic,
	}
	fn(nr)
}

func (jr *FastRouter) Handle(method, path string, handler Handler[rest.JSONResponse], opts ...Option) {
	handle(method, path, handler, jr, opts...)
}

func (jr *FastRouter) HandleAttachment(method, path string, handler Handler[rest.AttachmentResponse], opts ...Option) {
	handle(method, path, handler, jr, opts...)
}

func handle[T rest.Response](method, path string, handler Handler[T], jr *FastRouter, opts ...Option) {
	fullPath := jr.Options.Prefix + path
	jr.app.Add(method, fullPath, func(ctx *fiber.Ctx) error {
		timeout := jr.Options.RequestTimeout
		if ok, tOut := isUsedSpecificTimeout(opts...); ok {
			timeout = *tOut
		}

		if timeout > 0 {
			timeoutContext, cancel := context.WithTimeout(ctx.UserContext(), timeout)
			defer cancel()
			ctx.SetUserContext(timeoutContext)
		}

		req := newRequest(&requestOptions{
			Req:    ctx.Request(),
			Params: ctx.AllParams(),
		})

		var txn *newrelic.Transaction
		defer func() {
			if txn != nil {
				header := newrelicLib.TransformResponseHeaders(&ctx.Context().Response)
				rw := txn.SetWebResponse(&newrelicLib.ResponseWriter{
					HttpHeader: header,
				})

				rw.WriteHeader(ctx.Context().Response.StatusCode())

				txn.End()
			}
		}()

		if jr.newRelic != nil {
			txn = jr.newRelic.StartTransaction(ctx.Method() + " " + ctx.Path())
			txn.SetWebRequestHTTP(newrelicLib.ToHTTPRequest(ctx.Context()))
			ctx.SetUserContext(newrelic.NewContext(ctx.UserContext(), txn))
			ctx.SetUserContext(context.WithValue(ctx.UserContext(), tracing.NewRelicTransactionKey, txn))
		}

		sentryTxn := sentry.StartTransaction(ctx.UserContext(), ctx.Method()+" "+ctx.Path())
		defer sentryTxn.Finish()

		ctx.SetUserContext(sentryTxn.Context())

		var defOpts []Option
		// add default must authorize
		defOpts = append(defOpts, defaultMustAuthorized)
		defOpts = append(defOpts, opts...)

		if valid := isMustAuthorized(defOpts...); valid {

			roles, hasRoles := getRolesFromOptions(defOpts...)
			var authErr error
			if hasRoles {
				authErr = validateRequestContextWithRoles(ctx.UserContext(), roles)
			} else {
				authErr = validateRequestContext(ctx.UserContext())
			}

			if authErr != nil {
				return authErr
			}
		}

		respChan := make(chan handlerResult[T])

		go func() {
			result, err := panicHandler(handler)(ctx.UserContext(), req)
			respChan <- handlerResult[T]{
				Resp: result,
				Err:  err,
			}
		}()

		select {
		case <-ctx.UserContext().Done():
			if errors.Is(ctx.UserContext().Err(), context.DeadlineExceeded) {
				return fiber.ErrRequestTimeout
			}
		case resp := <-respChan:
			result := resp.Resp
			err := resp.Err

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return fiber.ErrRequestTimeout
				}
				// send to global error handler
				return err
			}

			if result == nil {
				// something went wrong when result is nil
				return &custerr.ErrChain{
					Message: "Internal server error",
					Code:    http.StatusInternalServerError,
					Type:    response.ErrInternalServerError,
				}
			}

			sendRes := *result
			return sendRes.Send(ctx)
		}

		// can't get result response, set to internal server error
		return &custerr.ErrChain{
			Message: "Internal server error",
			Code:    http.StatusInternalServerError,
			Type:    response.ErrInternalServerError,
		}
	})
}

func (jr *FastRouter) CustomHandler(method, path string, handler fiber.Handler, opts ...Option) {
	fullPath := jr.Options.Prefix + path
	jr.app.Add(method, fullPath, func(ctx *fiber.Ctx) error {
		timeout := jr.Options.RequestTimeout
		if ok, tOut := isUsedSpecificTimeout(opts...); ok {
			timeout = *tOut
		}

		if timeout > 0 {
			timeoutContext, cancel := context.WithTimeout(ctx.UserContext(), timeout)
			defer cancel()
			ctx.SetUserContext(timeoutContext)
		}

		var txn *newrelic.Transaction
		defer func() {
			if txn != nil {
				header := newrelicLib.TransformResponseHeaders(&ctx.Context().Response)
				rw := txn.SetWebResponse(&newrelicLib.ResponseWriter{
					HttpHeader: header,
				})

				rw.WriteHeader(ctx.Context().Response.StatusCode())

				txn.End()
			}
		}()

		if jr.newRelic != nil && filterAllowPath(ctx.Path()) {
			txn = jr.newRelic.StartTransaction(ctx.Method() + " " + ctx.Path())
			txn.SetWebRequestHTTP(newrelicLib.ToHTTPRequest(ctx.Context()))
			ctx.SetUserContext(newrelic.NewContext(ctx.UserContext(), txn))
			ctx.SetUserContext(context.WithValue(ctx.UserContext(), tracing.NewRelicTransactionKey, txn))
		}

		sentryTxn := sentry.StartTransaction(ctx.UserContext(), ctx.Method()+" "+ctx.Path())
		defer sentryTxn.Finish()

		ctx.SetUserContext(sentryTxn.Context())

		var defOpts []Option
		// add default must authorize
		defOpts = append(defOpts, defaultMustAuthorized)
		defOpts = append(defOpts, opts...)

		if valid := isMustAuthorized(defOpts...); valid {
			if err := validateRequestContext(ctx.UserContext()); err != nil {
				return err
			}
		}

		respChan := make(chan error)

		go func() {
			respChan <- panicHandlerFiber(ctx, handler)
		}()

		select {
		case <-ctx.UserContext().Done():
			return fiber.ErrRequestTimeout
		case err := <-respChan:
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return fiber.ErrRequestTimeout
				}
				// send to global error handler
				return err
			}
			return nil
		}
	})
}

func (jr *FastRouter) Test(req *http.Request, msTimeout ...int) (resp *http.Response, err error) {
	if len(msTimeout) == 0 {
		return jr.app.Test(req, int(jr.Options.RequestTimeout.Milliseconds()))
	}
	return jr.app.Test(req, msTimeout...)
}

func filterAllowPath(path string) bool {
	return !strings.Contains(path, "/docs/index.html") &&
		!strings.Contains(path, ".js") &&
		!strings.Contains(path, ".png") &&
		!strings.Contains(path, ".css")
}

func (jr *FastRouter) StartServe() error {
	return jr.app.Listen(fmt.Sprintf(":%d", jr.Options.Port))
}

func (jr *FastRouter) Shutdown() error {
	return jr.app.Shutdown()
}
