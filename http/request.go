package http

import (
	"context"
	"github.com/avast/retry-go/v4"
	"io"
	"time"
)

type RequestBuilder[T any] struct {
	request Request[T]
	client  Client
	retry   *RetryConfig
}

func NewRequest[T any](method Method, url string, target T, c Client) *RequestBuilder[T] {
	return &RequestBuilder[T]{
		request: Request[T]{
			Method:         method,
			URL:            url,
			TargetResponse: target,
			Headers:        make(map[string]string),
			URLParameters:  make(map[string]Parameter),
		},
		client: c,
	}
}

func NewGETRequest[T any](url string, target T, c Client) *RequestBuilder[T] {
	return NewRequest(GET, url, target, c)
}

func NewPOSTRequest[T any](url string, target T, c Client) *RequestBuilder[T] {
	return NewRequest(POST, url, target, c).WithBodyFormat(RawJson)
}

func NewPUTRequest[T any](url string, target T, c Client) *RequestBuilder[T] {
	return NewRequest(PUT, url, target, c).WithBodyFormat(RawJson)
}

func NewPATCHRequest[T any](url string, target T, c Client) *RequestBuilder[T] {
	return NewRequest(PATCH, url, target, c).WithBodyFormat(RawJson)
}

func NewDELETERequest[T any](url string, target T, c Client) *RequestBuilder[T] {
	return NewRequest(DELETE, url, target, c)
}

func (rb *RequestBuilder[T]) WithBody(body any) *RequestBuilder[T] {
	rb.request.RequestBody = body
	return rb
}

func (rb *RequestBuilder[T]) WithMultipartForm(forms map[string]io.Reader) *RequestBuilder[T] {
	rb.request.MultipartForm = forms
	return rb
}

func (rb *RequestBuilder[T]) WithContext(ctx context.Context) *RequestBuilder[T] {
	rb.request.Ctx = ctx
	return rb
}

func (rb *RequestBuilder[T]) AddHeader(key, value string) *RequestBuilder[T] {
	rb.request.Headers[key] = value
	return rb
}

func (rb *RequestBuilder[T]) WithHeaders(headers map[string]string) *RequestBuilder[T] {
	rb.request.Headers = headers
	return rb
}

func (rb *RequestBuilder[T]) WithBodyFormat(format BodyFormat) *RequestBuilder[T] {
	rb.request.BodyFormat = format
	return rb
}

func (rb *RequestBuilder[T]) WithParams(params map[string]Parameter) *RequestBuilder[T] {
	rb.request.URLParameters = params
	return rb
}

func (rb *RequestBuilder[T]) IsRawResponse(isRawResponse bool) *RequestBuilder[T] {
	rb.request.IsRawResponse = isRawResponse
	return rb
}

func (rb *RequestBuilder[T]) WithRetryConfig(config *RetryConfig) *RequestBuilder[T] {
	if config != nil && config.MaxRetry < 1 {
		// default only 1 retry
		config.MaxRetry = 1
	}
	rb.retry = config
	return rb
}

func (rb *RequestBuilder[T]) Execute() (*Response[T], error) {
	if rb.retry != nil {
		return rb.executeWithRetry()
	}
	return PerformRequest(rb.client, rb.request)
}

func (rb *RequestBuilder[T]) executeWithRetry() (resp *Response[T], err error) {
	resp = &Response[T]{}
	err = retry.Do(
		func() error {
			resp, err = PerformRequest(rb.client, rb.request)
			if err != nil {
				return err
			}
			return nil
		},
		retry.Attempts(uint(rb.retry.MaxRetry)),
		retry.DelayType(setDelayType(rb.retry.WithBackOff, rb.retry.MaxJitter)),
		retry.Delay(rb.retry.RetryInitialDelay),
		retry.MaxJitter(rb.retry.MaxJitter),
	)

	return resp, err
}

func setDelayType(withBackOff bool, maxJitter time.Duration) func(n uint, err error, config *retry.Config) time.Duration {
	return func(n uint, err error, config *retry.Config) time.Duration {
		if maxJitter > time.Millisecond {
			return retry.RandomDelay(n, err, config)
		}
		if withBackOff {
			return retry.BackOffDelay(n, err, config)
		}
		return retry.FixedDelay(n, err, config)
	}
}
