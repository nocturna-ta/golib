package sentry

import (
	"bytes"
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"golib/log"
	"io"
	"net/http"
	"net/url"
	"time"
)

type contextKey struct{}

var (
	ContextKey = contextKey{}
)

type (
	handler struct {
		waitForDelivery bool
		timeout         time.Duration
	}

	Config struct {
		MustStart bool
		// The DSN to use. If the DSN is not set, the client is effectively
		// disabled.
		Dsn string
		// In debug mode, the debug information is printed to stdout to help you
		// understand what sentry is doing.
		Debug bool
		// The sample rate for event submission in the range [0.0, 1.0]. By default,
		// all events are sent. Thus, as a historical special case, the sample rate
		// 0.0 is treated as if it was 1.0. To drop all events, set the DSN to the
		// empty string.
		SampleRate float64
		// Enable performance tracing.
		EnableTracing bool
		// The sample rate for sampling traces in the range [0.0, 1.0].
		TracesSampleRate float64
		// Used to customize the sampling of traces, overrides TracesSampleRate.
		TracesSampler sentry.TracesSampler
		// The server name to be reported.
		ServerName string
		// The environment to be sent with events.
		Environment string
	}

	Options struct {
		// WaitForDelivery configures whether you want to block the request before moving forward with the response.
		// Because fasthttp doesn't include its own Recovery handler, it will restart the application,
		// and event won't be delivered otherwise.
		WaitForDelivery bool
		// Timeout for the event delivery requests.
		Timeout time.Duration
	}
)

func Init(cfg *Config) error {
	opts := sentry.ClientOptions{
		Dsn:         cfg.Dsn,
		Environment: "development",
	}

	if cfg.Debug {
		opts.Debug = true
	}

	if cfg.SampleRate > 0 && cfg.SampleRate < 1.0 {
		opts.SampleRate = cfg.SampleRate
	}

	if cfg.EnableTracing {
		opts.EnableTracing = true
	}

	if cfg.TracesSampleRate > 0 {
		opts.TracesSampleRate = cfg.TracesSampleRate
	}

	if cfg.TracesSampler != nil {
		opts.TracesSampler = cfg.TracesSampler
	}

	if cfg.ServerName != "" {
		opts.ServerName = cfg.ServerName
	}

	if cfg.Environment != "" {
		opts.Environment = cfg.Environment
	}

	err := sentry.Init(opts)
	if err != nil {
		if cfg.MustStart {
			log.WithError(err).Fatal("Failed to start sentry, stopping")
		}
		return err
	}
	return nil
}

func New(options *Options) fiber.Handler {
	handler := handler{
		timeout:         time.Second * 2,
		waitForDelivery: false,
	}

	if options.Timeout != 0 {
		handler.timeout = options.Timeout
	}

	if options.WaitForDelivery {
		handler.waitForDelivery = true
	}

	return handler.handle
}

func (h *handler) handle(ctx *fiber.Ctx) error {
	hub := sentry.CurrentHub().Clone()
	scope := hub.Scope()
	scope.SetRequest(convert(ctx))
	scope.SetRequestBody(ctx.Request().Body())
	ctx.SetUserContext(context.WithValue(ctx.UserContext(), ContextKey, hub))
	defer h.recoverWithSentry(hub, ctx)
	return ctx.Next()
}

func (h *handler) recoverWithSentry(hub *sentry.Hub, ctx *fiber.Ctx) {
	if err := recover(); err != nil {
		eventID := hub.RecoverWithContext(
			context.WithValue(context.Background(), sentry.RequestContextKey, ctx),
			err,
		)
		if eventID != nil && h.waitForDelivery {
			hub.Flush(h.timeout)
		}
		panic(err)
	}
}

func GetHubFromContext(ctx context.Context) *sentry.Hub {
	hub := ctx.Value(ContextKey)
	if hub, ok := hub.(*sentry.Hub); ok {
		return hub
	}
	return nil
}

func convert(ctx *fiber.Ctx) *http.Request {
	defer func() {
		if err := recover(); err != nil {
			sentry.Logger.Printf("%v", err)
		}
	}()

	r := new(http.Request)

	r.Method = utils.CopyString(ctx.Method())
	uri := ctx.Request().URI()
	r.URL, _ = url.Parse(fmt.Sprintf("%s://%s%s", uri.Scheme(), uri.Host(), uri.Path()))

	// Headers
	r.Header = make(http.Header)
	ctx.Request().Header.VisitAll(func(key, value []byte) {
		r.Header.Add(string(key), string(value))
	})
	r.Host = utils.CopyString(ctx.Hostname())

	// Cookies
	ctx.Request().Header.VisitAllCookie(func(key, value []byte) {
		r.AddCookie(&http.Cookie{Name: string(key), Value: string(value)})
	})

	// Env
	r.RemoteAddr = ctx.Context().RemoteAddr().String()

	// QueryString
	r.URL.RawQuery = string(ctx.Request().URI().QueryString())

	// Body
	r.Body = io.NopCloser(bytes.NewReader(ctx.Request().Body()))

	return r
}
