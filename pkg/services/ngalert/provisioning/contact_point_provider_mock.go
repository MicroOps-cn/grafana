// Code generated by mockery v2.12.1. DO NOT EDIT.
// mockery --name ContactPointProvider --with-expecter --structname MockContactPointProvider --filename contact_point_provider_mock.go

package provisioning

import (
	context "context"

	definitions "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	mock "github.com/stretchr/testify/mock"

	testing "testing"
)

// MockContactPointProvider is an autogenerated mock type for the ContactPointProvider type
type MockContactPointProvider struct {
	mock.Mock
}

type MockContactPointProvider_Expecter struct {
	mock *mock.Mock
}

func (_m *MockContactPointProvider) EXPECT() *MockContactPointProvider_Expecter {
	return &MockContactPointProvider_Expecter{mock: &_m.Mock}
}

// GetContactPoints provides a mock function with given fields: ctx, orgID
func (_m *MockContactPointProvider) GetContactPoints(ctx context.Context, orgID int64) ([]definitions.EmbeddedContactPoint, error) {
	ret := _m.Called(ctx, orgID)

	var r0 []definitions.EmbeddedContactPoint
	if rf, ok := ret.Get(0).(func(context.Context, int64) []definitions.EmbeddedContactPoint); ok {
		r0 = rf(ctx, orgID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]definitions.EmbeddedContactPoint)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, orgID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockContactPointProvider_GetContactPoints_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetContactPoints'
type MockContactPointProvider_GetContactPoints_Call struct {
	*mock.Call
}

// GetContactPoints is a helper method to define mock.On call
//  - ctx context.Context
//  - orgID int64
func (_e *MockContactPointProvider_Expecter) GetContactPoints(ctx interface{}, orgID interface{}) *MockContactPointProvider_GetContactPoints_Call {
	return &MockContactPointProvider_GetContactPoints_Call{Call: _e.mock.On("GetContactPoints", ctx, orgID)}
}

func (_c *MockContactPointProvider_GetContactPoints_Call) Run(run func(ctx context.Context, orgID int64)) *MockContactPointProvider_GetContactPoints_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(int64))
	})
	return _c
}

func (_c *MockContactPointProvider_GetContactPoints_Call) Return(_a0 []definitions.EmbeddedContactPoint, _a1 error) *MockContactPointProvider_GetContactPoints_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

// NewMockContactPointProvider creates a new instance of MockContactPointProvider. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockContactPointProvider(t testing.TB) *MockContactPointProvider {
	mock := &MockContactPointProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
