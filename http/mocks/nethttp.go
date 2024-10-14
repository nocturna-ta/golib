package mocks

import (
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

type ClientNetHttp struct {
	mock.Mock
}

func (_m *ClientNetHttp) Do(req *http.Request) (*http.Response, error) {
	ret := _m.Called(req)

	var r0 *http.Response
	if rf, ok := ret.Get(0).(func(*http.Request) *http.Response); ok {
		r0 = rf(req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*http.Response)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*http.Request) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewClientNetHttp creates a new instance of ClientNetHttp. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewClientNetHttp(t testing.TB) *ClientNetHttp {
	mockClient := &ClientNetHttp{}
	mockClient.Mock.Test(t)

	t.Cleanup(func() { mockClient.AssertExpectations(t) })

	return mockClient
}
