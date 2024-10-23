// Code generated by mockery. DO NOT EDIT.

package pubsub

import (
	context "context"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/proto/eventbus/v1"
	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"
)

// mockPubSubClient is an autogenerated mock type for the PubSubClient type
type mockPubSubClient struct {
	mock.Mock
}

type mockPubSubClient_Expecter struct {
	mock *mock.Mock
}

func (_m *mockPubSubClient) EXPECT() *mockPubSubClient_Expecter {
	return &mockPubSubClient_Expecter{mock: &_m.Mock}
}

// GetSchema provides a mock function with given fields: ctx, in, opts
func (_m *mockPubSubClient) GetSchema(ctx context.Context, in *eventbusv1.SchemaRequest, opts ...grpc.CallOption) (*eventbusv1.SchemaInfo, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetSchema")
	}

	var r0 *eventbusv1.SchemaInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *eventbusv1.SchemaRequest, ...grpc.CallOption) (*eventbusv1.SchemaInfo, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *eventbusv1.SchemaRequest, ...grpc.CallOption) *eventbusv1.SchemaInfo); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*eventbusv1.SchemaInfo)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *eventbusv1.SchemaRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPubSubClient_GetSchema_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetSchema'
type mockPubSubClient_GetSchema_Call struct {
	*mock.Call
}

// GetSchema is a helper method to define mock.On call
//   - ctx context.Context
//   - in *eventbusv1.SchemaRequest
//   - opts ...grpc.CallOption
func (_e *mockPubSubClient_Expecter) GetSchema(ctx interface{}, in interface{}, opts ...interface{}) *mockPubSubClient_GetSchema_Call {
	return &mockPubSubClient_GetSchema_Call{Call: _e.mock.On("GetSchema",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *mockPubSubClient_GetSchema_Call) Run(run func(ctx context.Context, in *eventbusv1.SchemaRequest, opts ...grpc.CallOption)) *mockPubSubClient_GetSchema_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*eventbusv1.SchemaRequest), variadicArgs...)
	})
	return _c
}

func (_c *mockPubSubClient_GetSchema_Call) Return(_a0 *eventbusv1.SchemaInfo, _a1 error) *mockPubSubClient_GetSchema_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPubSubClient_GetSchema_Call) RunAndReturn(run func(context.Context, *eventbusv1.SchemaRequest, ...grpc.CallOption) (*eventbusv1.SchemaInfo, error)) *mockPubSubClient_GetSchema_Call {
	_c.Call.Return(run)
	return _c
}

// GetTopic provides a mock function with given fields: ctx, in, opts
func (_m *mockPubSubClient) GetTopic(ctx context.Context, in *eventbusv1.TopicRequest, opts ...grpc.CallOption) (*eventbusv1.TopicInfo, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for GetTopic")
	}

	var r0 *eventbusv1.TopicInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *eventbusv1.TopicRequest, ...grpc.CallOption) (*eventbusv1.TopicInfo, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *eventbusv1.TopicRequest, ...grpc.CallOption) *eventbusv1.TopicInfo); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*eventbusv1.TopicInfo)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *eventbusv1.TopicRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPubSubClient_GetTopic_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetTopic'
type mockPubSubClient_GetTopic_Call struct {
	*mock.Call
}

// GetTopic is a helper method to define mock.On call
//   - ctx context.Context
//   - in *eventbusv1.TopicRequest
//   - opts ...grpc.CallOption
func (_e *mockPubSubClient_Expecter) GetTopic(ctx interface{}, in interface{}, opts ...interface{}) *mockPubSubClient_GetTopic_Call {
	return &mockPubSubClient_GetTopic_Call{Call: _e.mock.On("GetTopic",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *mockPubSubClient_GetTopic_Call) Run(run func(ctx context.Context, in *eventbusv1.TopicRequest, opts ...grpc.CallOption)) *mockPubSubClient_GetTopic_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*eventbusv1.TopicRequest), variadicArgs...)
	})
	return _c
}

func (_c *mockPubSubClient_GetTopic_Call) Return(_a0 *eventbusv1.TopicInfo, _a1 error) *mockPubSubClient_GetTopic_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPubSubClient_GetTopic_Call) RunAndReturn(run func(context.Context, *eventbusv1.TopicRequest, ...grpc.CallOption) (*eventbusv1.TopicInfo, error)) *mockPubSubClient_GetTopic_Call {
	_c.Call.Return(run)
	return _c
}

// ManagedSubscribe provides a mock function with given fields: ctx, opts
func (_m *mockPubSubClient) ManagedSubscribe(ctx context.Context, opts ...grpc.CallOption) (eventbusv1.PubSub_ManagedSubscribeClient, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for ManagedSubscribe")
	}

	var r0 eventbusv1.PubSub_ManagedSubscribeClient
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) (eventbusv1.PubSub_ManagedSubscribeClient, error)); ok {
		return rf(ctx, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) eventbusv1.PubSub_ManagedSubscribeClient); ok {
		r0 = rf(ctx, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(eventbusv1.PubSub_ManagedSubscribeClient)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPubSubClient_ManagedSubscribe_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ManagedSubscribe'
type mockPubSubClient_ManagedSubscribe_Call struct {
	*mock.Call
}

// ManagedSubscribe is a helper method to define mock.On call
//   - ctx context.Context
//   - opts ...grpc.CallOption
func (_e *mockPubSubClient_Expecter) ManagedSubscribe(ctx interface{}, opts ...interface{}) *mockPubSubClient_ManagedSubscribe_Call {
	return &mockPubSubClient_ManagedSubscribe_Call{Call: _e.mock.On("ManagedSubscribe",
		append([]interface{}{ctx}, opts...)...)}
}

func (_c *mockPubSubClient_ManagedSubscribe_Call) Run(run func(ctx context.Context, opts ...grpc.CallOption)) *mockPubSubClient_ManagedSubscribe_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), variadicArgs...)
	})
	return _c
}

func (_c *mockPubSubClient_ManagedSubscribe_Call) Return(_a0 eventbusv1.PubSub_ManagedSubscribeClient, _a1 error) *mockPubSubClient_ManagedSubscribe_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPubSubClient_ManagedSubscribe_Call) RunAndReturn(run func(context.Context, ...grpc.CallOption) (eventbusv1.PubSub_ManagedSubscribeClient, error)) *mockPubSubClient_ManagedSubscribe_Call {
	_c.Call.Return(run)
	return _c
}

// Publish provides a mock function with given fields: ctx, in, opts
func (_m *mockPubSubClient) Publish(ctx context.Context, in *eventbusv1.PublishRequest, opts ...grpc.CallOption) (*eventbusv1.PublishResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, in)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Publish")
	}

	var r0 *eventbusv1.PublishResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *eventbusv1.PublishRequest, ...grpc.CallOption) (*eventbusv1.PublishResponse, error)); ok {
		return rf(ctx, in, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *eventbusv1.PublishRequest, ...grpc.CallOption) *eventbusv1.PublishResponse); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*eventbusv1.PublishResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *eventbusv1.PublishRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPubSubClient_Publish_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Publish'
type mockPubSubClient_Publish_Call struct {
	*mock.Call
}

// Publish is a helper method to define mock.On call
//   - ctx context.Context
//   - in *eventbusv1.PublishRequest
//   - opts ...grpc.CallOption
func (_e *mockPubSubClient_Expecter) Publish(ctx interface{}, in interface{}, opts ...interface{}) *mockPubSubClient_Publish_Call {
	return &mockPubSubClient_Publish_Call{Call: _e.mock.On("Publish",
		append([]interface{}{ctx, in}, opts...)...)}
}

func (_c *mockPubSubClient_Publish_Call) Run(run func(ctx context.Context, in *eventbusv1.PublishRequest, opts ...grpc.CallOption)) *mockPubSubClient_Publish_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), args[1].(*eventbusv1.PublishRequest), variadicArgs...)
	})
	return _c
}

func (_c *mockPubSubClient_Publish_Call) Return(_a0 *eventbusv1.PublishResponse, _a1 error) *mockPubSubClient_Publish_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPubSubClient_Publish_Call) RunAndReturn(run func(context.Context, *eventbusv1.PublishRequest, ...grpc.CallOption) (*eventbusv1.PublishResponse, error)) *mockPubSubClient_Publish_Call {
	_c.Call.Return(run)
	return _c
}

// PublishStream provides a mock function with given fields: ctx, opts
func (_m *mockPubSubClient) PublishStream(ctx context.Context, opts ...grpc.CallOption) (eventbusv1.PubSub_PublishStreamClient, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for PublishStream")
	}

	var r0 eventbusv1.PubSub_PublishStreamClient
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) (eventbusv1.PubSub_PublishStreamClient, error)); ok {
		return rf(ctx, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) eventbusv1.PubSub_PublishStreamClient); ok {
		r0 = rf(ctx, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(eventbusv1.PubSub_PublishStreamClient)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPubSubClient_PublishStream_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PublishStream'
type mockPubSubClient_PublishStream_Call struct {
	*mock.Call
}

// PublishStream is a helper method to define mock.On call
//   - ctx context.Context
//   - opts ...grpc.CallOption
func (_e *mockPubSubClient_Expecter) PublishStream(ctx interface{}, opts ...interface{}) *mockPubSubClient_PublishStream_Call {
	return &mockPubSubClient_PublishStream_Call{Call: _e.mock.On("PublishStream",
		append([]interface{}{ctx}, opts...)...)}
}

func (_c *mockPubSubClient_PublishStream_Call) Run(run func(ctx context.Context, opts ...grpc.CallOption)) *mockPubSubClient_PublishStream_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), variadicArgs...)
	})
	return _c
}

func (_c *mockPubSubClient_PublishStream_Call) Return(_a0 eventbusv1.PubSub_PublishStreamClient, _a1 error) *mockPubSubClient_PublishStream_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPubSubClient_PublishStream_Call) RunAndReturn(run func(context.Context, ...grpc.CallOption) (eventbusv1.PubSub_PublishStreamClient, error)) *mockPubSubClient_PublishStream_Call {
	_c.Call.Return(run)
	return _c
}

// Subscribe provides a mock function with given fields: ctx, opts
func (_m *mockPubSubClient) Subscribe(ctx context.Context, opts ...grpc.CallOption) (eventbusv1.PubSub_SubscribeClient, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Subscribe")
	}

	var r0 eventbusv1.PubSub_SubscribeClient
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) (eventbusv1.PubSub_SubscribeClient, error)); ok {
		return rf(ctx, opts...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) eventbusv1.PubSub_SubscribeClient); ok {
		r0 = rf(ctx, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(eventbusv1.PubSub_SubscribeClient)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockPubSubClient_Subscribe_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Subscribe'
type mockPubSubClient_Subscribe_Call struct {
	*mock.Call
}

// Subscribe is a helper method to define mock.On call
//   - ctx context.Context
//   - opts ...grpc.CallOption
func (_e *mockPubSubClient_Expecter) Subscribe(ctx interface{}, opts ...interface{}) *mockPubSubClient_Subscribe_Call {
	return &mockPubSubClient_Subscribe_Call{Call: _e.mock.On("Subscribe",
		append([]interface{}{ctx}, opts...)...)}
}

func (_c *mockPubSubClient_Subscribe_Call) Run(run func(ctx context.Context, opts ...grpc.CallOption)) *mockPubSubClient_Subscribe_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]grpc.CallOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(grpc.CallOption)
			}
		}
		run(args[0].(context.Context), variadicArgs...)
	})
	return _c
}

func (_c *mockPubSubClient_Subscribe_Call) Return(_a0 eventbusv1.PubSub_SubscribeClient, _a1 error) *mockPubSubClient_Subscribe_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockPubSubClient_Subscribe_Call) RunAndReturn(run func(context.Context, ...grpc.CallOption) (eventbusv1.PubSub_SubscribeClient, error)) *mockPubSubClient_Subscribe_Call {
	_c.Call.Return(run)
	return _c
}

// newMockPubSubClient creates a new instance of mockPubSubClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockPubSubClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockPubSubClient {
	mock := &mockPubSubClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
