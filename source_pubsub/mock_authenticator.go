// Code generated by mockery v2.43.0. DO NOT EDIT.

package source

import mock "github.com/stretchr/testify/mock"

// mockAuthenticator is an autogenerated mock type for the authenticator type
type mockAuthenticator struct {
	mock.Mock
}

type mockAuthenticator_Expecter struct {
	mock *mock.Mock
}

func (_m *mockAuthenticator) EXPECT() *mockAuthenticator_Expecter {
	return &mockAuthenticator_Expecter{mock: &_m.Mock}
}

// Login provides a mock function with given fields:
func (_m *mockAuthenticator) Login() (*LoginResponse, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Login")
	}

	var r0 *LoginResponse
	var r1 error
	if rf, ok := ret.Get(0).(func() (*LoginResponse, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *LoginResponse); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*LoginResponse)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockAuthenticator_Login_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Login'
type mockAuthenticator_Login_Call struct {
	*mock.Call
}

// Login is a helper method to define mock.On call
func (_e *mockAuthenticator_Expecter) Login() *mockAuthenticator_Login_Call {
	return &mockAuthenticator_Login_Call{Call: _e.mock.On("Login")}
}

func (_c *mockAuthenticator_Login_Call) Run(run func()) *mockAuthenticator_Login_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockAuthenticator_Login_Call) Return(_a0 *LoginResponse, _a1 error) *mockAuthenticator_Login_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockAuthenticator_Login_Call) RunAndReturn(run func() (*LoginResponse, error)) *mockAuthenticator_Login_Call {
	_c.Call.Return(run)
	return _c
}

// UserInfo provides a mock function with given fields: _a0
func (_m *mockAuthenticator) UserInfo(_a0 string) (*UserInfoResponse, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for UserInfo")
	}

	var r0 *UserInfoResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*UserInfoResponse, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(string) *UserInfoResponse); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*UserInfoResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockAuthenticator_UserInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UserInfo'
type mockAuthenticator_UserInfo_Call struct {
	*mock.Call
}

// UserInfo is a helper method to define mock.On call
//   - _a0 string
func (_e *mockAuthenticator_Expecter) UserInfo(_a0 interface{}) *mockAuthenticator_UserInfo_Call {
	return &mockAuthenticator_UserInfo_Call{Call: _e.mock.On("UserInfo", _a0)}
}

func (_c *mockAuthenticator_UserInfo_Call) Run(run func(_a0 string)) *mockAuthenticator_UserInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockAuthenticator_UserInfo_Call) Return(_a0 *UserInfoResponse, _a1 error) *mockAuthenticator_UserInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockAuthenticator_UserInfo_Call) RunAndReturn(run func(string) (*UserInfoResponse, error)) *mockAuthenticator_UserInfo_Call {
	_c.Call.Return(run)
	return _c
}

// newMockAuthenticator creates a new instance of mockAuthenticator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockAuthenticator(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockAuthenticator {
	mock := &mockAuthenticator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
