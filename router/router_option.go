package router

import "time"

var (
	defaultMustAuthorized = MustAuthorized(true)
)

type option struct {
	mustAuthorized bool
	requestTimeout *time.Duration
}

type Option interface {
	Apply(opt *option)
}

type OptionFn func(opt *option)

func (of OptionFn) Apply(opt *option) {
	of(opt)
}

// MustAuthorized option will set the authorization checking
// set to true if you need to make access to route must be authorized
func MustAuthorized(val bool) OptionFn {
	return func(opt *option) {
		opt.mustAuthorized = val
	}
}

// WithTimeout set specific timeout for specific endpoint
func WithTimeout(val time.Duration) OptionFn {
	return func(opt *option) {
		opt.requestTimeout = &val
	}
}

func isMustAuthorized(opts ...Option) bool {
	opt := &option{
		mustAuthorized: false,
	}
	for _, op := range opts {
		op.Apply(opt)
	}
	return opt.mustAuthorized
}

func isUsedSpecificTimeout(opts ...Option) (bool, *time.Duration) {
	opt := &option{}
	for _, op := range opts {
		op.Apply(opt)
	}
	return opt.requestTimeout != nil, opt.requestTimeout
}
