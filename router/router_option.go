package router

import "time"

var (
	defaultMustAuthorized = MustAuthorized(true)
)

type option struct {
	mustAuthorized bool
	requestTimeout *time.Duration
	allowedRoles   []string
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

func WithRoles(roles ...string) OptionFn {
	return func(opt *option) {
		opt.mustAuthorized = true
		opt.allowedRoles = roles
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

func getRolesFromOptions(opts ...Option) ([]string, bool) {
	opt := &option{}
	for _, op := range opts {
		op.Apply(opt)
	}
	return opt.allowedRoles, len(opt.allowedRoles) > 0
}
