package newrelic

import (
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/nocturna-ta/golib/log"
	"github.com/valyala/fasthttp"
	"net/http"
	"net/url"
)

// ResponseWriter imitates http.ResponseWriter
type ResponseWriter struct {
	StatusCode int
	HttpHeader http.Header
	Body       []byte
}

type Options struct {
	MustStart bool
}

// Header implementation
func (rw *ResponseWriter) Header() http.Header {
	if rw.HttpHeader == nil {
		rw.HttpHeader = make(http.Header)
	}
	return rw.HttpHeader
}

// WriteHeader implementation
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
}

// Write implementation
func (rw *ResponseWriter) Write(p []byte) (int, error) {
	rw.Body = append(rw.Body, p...)
	return len(p), nil
}

func SetupNewRelic(opts *Options) *newrelic.Application {
	if opts == nil {
		return nil
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
	)
	if err != nil {
		if opts.MustStart {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Failed to start new relic application")
		}
		return nil
	}

	return app
}

func ToHTTPRequest(ctx *fasthttp.RequestCtx) *http.Request {
	uri := ctx.Request.URI()
	url := &url.URL{
		Scheme:   string(uri.Scheme()),
		Path:     string(uri.Path()),
		Host:     string(uri.Host()),
		RawQuery: string(uri.QueryString()),
	}

	return &http.Request{
		Method: string(ctx.Request.Header.Method()),
		URL:    url,
		Proto:  "HTTP/1.1",
		Header: transformRequestHeaders(&ctx.Request),
		Host:   string(uri.Host()),
		TLS:    ctx.TLSConnectionState(),
	}
}

func transformRequestHeaders(r *fasthttp.Request) http.Header {
	header := make(http.Header)
	r.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)
		header.Set(sk, sv)
	})

	return header
}

func TransformResponseHeaders(r *fasthttp.Response) http.Header {
	header := make(http.Header)
	r.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)
		header.Set(sk, sv)
	})

	return header
}
