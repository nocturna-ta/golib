package http

import (
	"context"
	"io"
	"net/http"
	"time"
)

type httpConfig struct {
	BaseUrl string
	Timeout time.Duration
}

type Request[T any] struct {
	Method         Method
	URL            string
	RequestBody    any
	TargetResponse T
	URLParameters  map[string]Parameter
	Headers        map[string]string
	BodyFormat     BodyFormat
	MultipartForm  map[string]io.Reader
	IsRawResponse  bool
	Ctx            context.Context
}

type Response[T any] struct {
	HttpCode       int
	ResponseHeader http.Header
	ResponseData   T
	RawResponse    []byte
	Method         Method
	URL            string
}

type RetryConfig struct {
	MaxRetry          int
	RetryInitialDelay time.Duration
	MaxJitter         time.Duration // set MaxJitter > 1ms will enable random retry and will discard WithBackOff value
	WithBackOff       bool          // indicate if we need to use delay with back-off mechanism, discard if set MaxJitter
}

type File struct {
	io.Reader
	FileName string
}

// Parameter is used to provide URL Parameters to nethttp client request
type Parameter string

// BodyFormat is used to define what request body format that need to send
type BodyFormat string

// Method is a type for defining the possible HTTP request methods that can be used
type Method string

const (
	// GET is the GET method for clienthttp requests
	GET Method = "GET"
	// POST is the POST method for clienthttp requests
	POST Method = "POST"
	// PUT is the PUT method for clienthttp requests
	PUT Method = "PUT"
	// PATCH is the PUT method for clienthttp requests
	PATCH Method = "PATCH"
	// DELETE is the DELETE method for clienthttp requests
	DELETE Method = "DELETE"
)

const (
	Raw           BodyFormat = "raw"
	RawFile       BodyFormat = "raw-file"
	RawJson       BodyFormat = "json"
	FormJsonData  BodyFormat = "jsondata"
	MultipartForm BodyFormat = "multipart-form"
	EmptyBody     BodyFormat = ""
)
