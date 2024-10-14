package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	libCtx "golib/context"
	"golib/log"
	"golib/tracing"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"time"
)

type ClientNetHttp interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	config httpConfig
	client ClientNetHttp
}

type Options struct {
	BaseUrl string
	Timeout time.Duration
}

// NewHttpClient
// create new default custom http client
func NewHttpClient(opts *Options) Client {
	return NewHttpWithClient(opts, &http.Client{
		Timeout: opts.Timeout,
	})
}

// NewHttpWithClient
// create new custom http client with given client c ClientNetHttp
func NewHttpWithClient(opts *Options, c ClientNetHttp) Client {
	return Client{
		config: httpConfig{
			BaseUrl: opts.BaseUrl,
			Timeout: opts.Timeout,
		},
		client: c,
	}
}

func PerformRequest[T any](c Client, httpRequest Request[T]) (httpResponse *Response[T], err error) {
	span, ctx := tracing.StartSpanFromContext(httpRequest.Ctx, "ClientNetHttp.PerformRequest")
	defer span.End()

	httpRequest.Ctx = ctx

	if !httpRequest.IsRawResponse {
		if reflect.TypeOf(httpRequest.TargetResponse).Kind() != reflect.Ptr {
			return nil, fmt.Errorf("failed, you must pass a pointer in the TargetResponse field of httpRequest")
		}
	}

	// Retrieve URL parameters
	endpointURL := c.config.BaseUrl + httpRequest.URL
	q := url.Values{}
	for k, v := range httpRequest.URLParameters {
		if v != "" {
			q.Set(k, string(v))
		}
	}

	var (
		reqBody     io.Reader
		contentType string
	)

	// default content-type
	contentType = "application/json;charset=utf-8"

	if httpRequest.BodyFormat != EmptyBody {
		switch httpRequest.BodyFormat {
		case RawFile:
			if x, ok := httpRequest.RequestBody.(io.Reader); ok {
				xBody, err := io.ReadAll(x)
				if err != nil {
					return nil, err
				}
				reqBody = bytes.NewBuffer(xBody)
				contentType = "multipart/form-data"
			}
		case Raw:
			if rawBody, ok := httpRequest.RequestBody.(string); ok {
				reqBody = bytes.NewBuffer([]byte(rawBody))
				contentType = "text/plain"
			}
		case RawJson:
			// JSON Marshal the body
			marshalledBody, err := json.Marshal(httpRequest.RequestBody)
			if err != nil {
				return nil, fmt.Errorf("failed to create json from request body")
			}

			reqBody = bytes.NewReader(marshalledBody)
			contentType = "application/json;charset=utf-8"
		case FormJsonData:
			// Create a multipart form
			var b bytes.Buffer
			w := multipart.NewWriter(&b)

			// Use the form to create the proper field
			fw, err := w.CreateFormField("jsondata")
			if err != nil {
				return nil, err
			}
			// Copy the request body JSON into the field
			if _, err = io.Copy(fw, reqBody); err != nil {
				return nil, err
			}

			// Close the multipart writer to set the terminating boundary
			err = w.Close()
			if err != nil {
				return nil, err
			}

			reqBody = &b
			w.Close()
			contentType = w.FormDataContentType()
		case MultipartForm:
			var b bytes.Buffer
			w := multipart.NewWriter(&b)

			for key, r := range httpRequest.MultipartForm {
				var fw io.Writer
				if x, ok := r.(io.Closer); ok {
					defer x.Close()
				}
				// Add a file
				if x, ok := r.(*os.File); ok {
					if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
						return nil, err
					}
				} else if y, ok := r.(*File); ok {
					if fw, err = w.CreateFormFile(key, y.FileName); err != nil {
						return nil, err
					}
				} else {
					// Add other fields
					if fw, err = w.CreateFormField(key); err != nil {
						return nil, err
					}
				}
				if _, err = io.Copy(fw, r); err != nil {
					return nil, err
				}
			}

			// Close the multipart writer to set the terminating boundary
			err = w.Close()
			if err != nil {
				return nil, err
			}

			reqBody = &b
			w.Close()
			contentType = w.FormDataContentType()
		}
	}

	encodedQuery := q.Encode()
	if encodedQuery != "" {
		endpointURL = fmt.Sprintf("%s?%s", endpointURL, q.Encode())
	}

	req, err := http.NewRequestWithContext(httpRequest.Ctx, string(httpRequest.Method), endpointURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create a request for %s: %s", httpRequest.URL, err)
	}

	req.Header.Add("Content-Type", contentType)

	addHeaderFromCtx(httpRequest.Ctx, req)

	// Add specific endpoint headers
	for k, v := range httpRequest.Headers {
		req.Header.Add(k, v)
	}

	resp, err := c.client.Do(req)
	logFields := log.Fields{
		"error": err,
	}
	if resp != nil {
		logFields["request"] = resp.Request
		logFields["response-status"] = resp.Status
		logFields["response-code"] = resp.StatusCode
		logFields["response-header"] = resp.Header
	}
	log.WithFields(logFields).DebugWithCtx(ctx, "[http/perform] Response from do request")
	if err != nil {
		return nil, fmt.Errorf("failed to perform request for %s: %s", httpRequest.URL, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warnf("Failed to close request body: %s\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body of response for %s: got status %s: %s", httpRequest.URL, resp.Status, err)
	}

	httpResponse = &Response[T]{
		HttpCode:       resp.StatusCode,
		ResponseHeader: resp.Header,
		Method:         Method(req.Method),
		URL:            endpointURL,
	}

	if httpRequest.IsRawResponse {
		httpResponse.RawResponse = body
		return httpResponse, nil
	}

	err = json.Unmarshal(body, &httpResponse.ResponseData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data from response for %s: got status %s: %s", httpRequest.URL, resp.Status, err)
	}

	return httpResponse, nil
}

func addHeaderFromCtx(ctx context.Context, req *http.Request) {
	reqCtx, err := libCtx.GetRequestContext(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Failed to get request ctx")
		return
	}

	req.Header.Add(libCtx.XRequestId, reqCtx.RequestId)

	if reqCtx.UserId != "" {
		req.Header.Add(libCtx.XUserId, reqCtx.UserId)
	}

	if reqCtx.ChannelId != "" {
		req.Header.Add(libCtx.XChannelId, reqCtx.ChannelId)
	}

	if reqCtx.AccountId != "" {
		req.Header.Add(libCtx.XAccountId, reqCtx.AccountId)
	}
}
