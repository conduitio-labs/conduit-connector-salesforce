// Code generated by mockery. DO NOT EDIT.

package pubsub

import (
	context "context"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/proto/eventbus/v1"
	metadata "google.golang.org/grpc/metadata"

	mock "github.com/stretchr/testify/mock"
)

// mockPubSub_SubscribeClient is an autogenerated mock type for the PubSub_SubscribeClient type
type mockPubSub_SubscribeClient struct {
	mock.Mock
}

type mockPubSub_SubscribeClient_Expecter struct {
	mock *mock.Mock
}

func (_m *mockPubSub_SubscribeClient) EXPECT() *mockPubSub_SubscribeClient_Expecter {
	return &mockPubSub_SubscribeClient_Expecter{mock: &_m.Mock}
}

// CloseSend provides a mock function with no fields
func (_m *mockPubSub_SubscribeClient) CloseSend() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for CloseSend")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPubSub_SubscribeClient_CloseSend_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CloseSend'
type mockPubSub_SubscribeClient_CloseSend_Call struct {
	*mock.Call
}

// CloseSend is a helper method to define mock.On call
func (_e *mockPubSub_SubscribeClient_Expecter) CloseSend() *mockPubSub_SubscribeClient_CloseSend_Call {
	return &mockPubSub_SubscribeClient_CloseSend_Call{Call: _e.mock.On("CloseSend")}
}

func (_c *mockPubSub_SubscribeClient_CloseSend_Call) Run(run func()) *mockPubSub_SubscribeClient_CloseSend_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockPubSub_SubscribeClient_CloseSend_Call) Return(_a0 error) *mockPubSub_SubscribeClient_CloseSend_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPubSub_SubscribeClient_CloseSend_Call) RunAndReturn(run func() error) *mockPubSub_SubscribeClient_CloseSend_Call {
	_c.Call.Return(run)
	return _c
}

// Context provides a mock function with no fields
func (_m *mockPubSub_SubscribeClient) Context() context.Context {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Context")
	}

	var r0 context.Context
	if rf, ok := ret.Get(0).(func() context.Context); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(context.Context)
		}
	}

	return r0
}

// mockPubSub_SubscribeClient_Context_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Context'
type mockPubSub_SubscribeClient_Context_Call struct {
	*mock.Call
}

// Context is a helper method to define mock.On call
func (_e *mockPubSub_SubscribeClient_Expecter) Context() *mockPubSub_SubscribeClient_Context_Call {
	return &mockPubSub_SubscribeClient_Context_Call{Call: _e.mock.On("Context")}
}

func (_c *mockPubSub_SubscribeClient_Context_Call) Run(run func()) *mockPubSub_SubscribeClient_Context_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockPubSub_SubscribeClient_Context_Call) Return(_a0 context.Context) *mockPubSub_SubscribeClient_Context_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPubSub_SubscribeClient_Context_Call) RunAndReturn(run func() context.Context) *mockPubSub_SubscribeClient_Context_Call {
	_c.Call.Return(run)
	return _c
}

// Header provides a mock function with no fields
func (_m *mockPubSub_SubscribeClient) Header() (metadata.MD, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Header")
	}

	var r0 metadata.MD
	var r1 error
	if rf, ok := ret.Get(0).(func() (metadata.MD, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() metadata.MD); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metadata.MD)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPubSub_SubscribeClient_Header_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Header'
type mockPubSub_SubscribeClient_Header_Call struct {
	*mock.Call
}

// Header is a helper method to define mock.On call
func (_e *mockPubSub_SubscribeClient_Expecter) Header() *mockPubSub_SubscribeClient_Header_Call {
	return &mockPubSub_SubscribeClient_Header_Call{Call: _e.mock.On("Header")}
}

func (_c *mockPubSub_SubscribeClient_Header_Call) Run(run func()) *mockPubSub_SubscribeClient_Header_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockPubSub_SubscribeClient_Header_Call) Return(_a0 metadata.MD, _a1 error) *mockPubSub_SubscribeClient_Header_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPubSub_SubscribeClient_Header_Call) RunAndReturn(run func() (metadata.MD, error)) *mockPubSub_SubscribeClient_Header_Call {
	_c.Call.Return(run)
	return _c
}

// Recv provides a mock function with no fields
func (_m *mockPubSub_SubscribeClient) Recv() (*eventbusv1.FetchResponse, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Recv")
	}

	var r0 *eventbusv1.FetchResponse
	var r1 error
	if rf, ok := ret.Get(0).(func() (*eventbusv1.FetchResponse, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *eventbusv1.FetchResponse); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*eventbusv1.FetchResponse)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPubSub_SubscribeClient_Recv_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Recv'
type mockPubSub_SubscribeClient_Recv_Call struct {
	*mock.Call
}

// Recv is a helper method to define mock.On call
func (_e *mockPubSub_SubscribeClient_Expecter) Recv() *mockPubSub_SubscribeClient_Recv_Call {
	return &mockPubSub_SubscribeClient_Recv_Call{Call: _e.mock.On("Recv")}
}

func (_c *mockPubSub_SubscribeClient_Recv_Call) Run(run func()) *mockPubSub_SubscribeClient_Recv_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockPubSub_SubscribeClient_Recv_Call) Return(_a0 *eventbusv1.FetchResponse, _a1 error) *mockPubSub_SubscribeClient_Recv_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPubSub_SubscribeClient_Recv_Call) RunAndReturn(run func() (*eventbusv1.FetchResponse, error)) *mockPubSub_SubscribeClient_Recv_Call {
	_c.Call.Return(run)
	return _c
}

// RecvMsg provides a mock function with given fields: m
func (_m *mockPubSub_SubscribeClient) RecvMsg(m interface{}) error {
	ret := _m.Called(m)

	if len(ret) == 0 {
		panic("no return value specified for RecvMsg")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(m)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPubSub_SubscribeClient_RecvMsg_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RecvMsg'
type mockPubSub_SubscribeClient_RecvMsg_Call struct {
	*mock.Call
}

// RecvMsg is a helper method to define mock.On call
//   - m interface{}
func (_e *mockPubSub_SubscribeClient_Expecter) RecvMsg(m interface{}) *mockPubSub_SubscribeClient_RecvMsg_Call {
	return &mockPubSub_SubscribeClient_RecvMsg_Call{Call: _e.mock.On("RecvMsg", m)}
}

func (_c *mockPubSub_SubscribeClient_RecvMsg_Call) Run(run func(m interface{})) *mockPubSub_SubscribeClient_RecvMsg_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *mockPubSub_SubscribeClient_RecvMsg_Call) Return(_a0 error) *mockPubSub_SubscribeClient_RecvMsg_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPubSub_SubscribeClient_RecvMsg_Call) RunAndReturn(run func(interface{}) error) *mockPubSub_SubscribeClient_RecvMsg_Call {
	_c.Call.Return(run)
	return _c
}

// Send provides a mock function with given fields: _a0
func (_m *mockPubSub_SubscribeClient) Send(_a0 *eventbusv1.FetchRequest) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Send")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*eventbusv1.FetchRequest) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPubSub_SubscribeClient_Send_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Send'
type mockPubSub_SubscribeClient_Send_Call struct {
	*mock.Call
}

// Send is a helper method to define mock.On call
//   - _a0 *eventbusv1.FetchRequest
func (_e *mockPubSub_SubscribeClient_Expecter) Send(_a0 interface{}) *mockPubSub_SubscribeClient_Send_Call {
	return &mockPubSub_SubscribeClient_Send_Call{Call: _e.mock.On("Send", _a0)}
}

func (_c *mockPubSub_SubscribeClient_Send_Call) Run(run func(_a0 *eventbusv1.FetchRequest)) *mockPubSub_SubscribeClient_Send_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*eventbusv1.FetchRequest))
	})
	return _c
}

func (_c *mockPubSub_SubscribeClient_Send_Call) Return(_a0 error) *mockPubSub_SubscribeClient_Send_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPubSub_SubscribeClient_Send_Call) RunAndReturn(run func(*eventbusv1.FetchRequest) error) *mockPubSub_SubscribeClient_Send_Call {
	_c.Call.Return(run)
	return _c
}

// SendMsg provides a mock function with given fields: m
func (_m *mockPubSub_SubscribeClient) SendMsg(m interface{}) error {
	ret := _m.Called(m)

	if len(ret) == 0 {
		panic("no return value specified for SendMsg")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(m)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockPubSub_SubscribeClient_SendMsg_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SendMsg'
type mockPubSub_SubscribeClient_SendMsg_Call struct {
	*mock.Call
}

// SendMsg is a helper method to define mock.On call
//   - m interface{}
func (_e *mockPubSub_SubscribeClient_Expecter) SendMsg(m interface{}) *mockPubSub_SubscribeClient_SendMsg_Call {
	return &mockPubSub_SubscribeClient_SendMsg_Call{Call: _e.mock.On("SendMsg", m)}
}

func (_c *mockPubSub_SubscribeClient_SendMsg_Call) Run(run func(m interface{})) *mockPubSub_SubscribeClient_SendMsg_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *mockPubSub_SubscribeClient_SendMsg_Call) Return(_a0 error) *mockPubSub_SubscribeClient_SendMsg_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPubSub_SubscribeClient_SendMsg_Call) RunAndReturn(run func(interface{}) error) *mockPubSub_SubscribeClient_SendMsg_Call {
	_c.Call.Return(run)
	return _c
}

// Trailer provides a mock function with no fields
func (_m *mockPubSub_SubscribeClient) Trailer() metadata.MD {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Trailer")
	}

	var r0 metadata.MD
	if rf, ok := ret.Get(0).(func() metadata.MD); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metadata.MD)
		}
	}

	return r0
}

// mockPubSub_SubscribeClient_Trailer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Trailer'
type mockPubSub_SubscribeClient_Trailer_Call struct {
	*mock.Call
}

// Trailer is a helper method to define mock.On call
func (_e *mockPubSub_SubscribeClient_Expecter) Trailer() *mockPubSub_SubscribeClient_Trailer_Call {
	return &mockPubSub_SubscribeClient_Trailer_Call{Call: _e.mock.On("Trailer")}
}

func (_c *mockPubSub_SubscribeClient_Trailer_Call) Run(run func()) *mockPubSub_SubscribeClient_Trailer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockPubSub_SubscribeClient_Trailer_Call) Return(_a0 metadata.MD) *mockPubSub_SubscribeClient_Trailer_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockPubSub_SubscribeClient_Trailer_Call) RunAndReturn(run func() metadata.MD) *mockPubSub_SubscribeClient_Trailer_Call {
	_c.Call.Return(run)
	return _c
}

// newMockPubSub_SubscribeClient creates a new instance of mockPubSub_SubscribeClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockPubSub_SubscribeClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockPubSub_SubscribeClient {
	mock := &mockPubSub_SubscribeClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
