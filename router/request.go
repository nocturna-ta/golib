package router

import (
	"github.com/valyala/fasthttp"
	"unsafe"
)

type (
	Request struct {
		req    *fasthttp.Request
		params map[string]string
	}

	requestOptions struct {
		Req    *fasthttp.Request
		Params map[string]string
	}
)

func newRequest(opt *requestOptions) *Request {
	return &Request{
		req:    opt.Req,
		params: opt.Params,
	}
}

func (r *Request) RawRequest() *fasthttp.Request {
	return r.req
}

func (r *Request) RawBody() []byte {
	return r.req.Body()
}

func (r *Request) Params(key string, defaultValue ...string) string {
	val, ok := r.params[key]
	if !ok {
		return defaultString("", defaultValue)
	}
	return val
}

func (r *Request) Query(key string, defaultValue ...string) string {
	return defaultString(byteToString(r.req.URI().QueryArgs().Peek(key)), defaultValue)
}

func (r *Request) Header(key string, defaultValue ...string) string {
	return defaultString(byteToString(r.req.Header.Peek(key)), defaultValue)
}

func byteToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func defaultString(value string, defaultValue []string) string {
	if len(value) == 0 && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return value
}
