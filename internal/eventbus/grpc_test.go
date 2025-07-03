// Copyright © 2025 Meroxa, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eventbus

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_streamInterceptor(t *testing.T) {
	ctx := context.TODO()
	is := is.New(t)
	tests := []struct {
		name     string
		grpcExt  func(t *testing.T) *grpcClientExt
		streamer grpc.Streamer
		wantErr  error
	}{
		{
			name: "succeeds creating stream",
			grpcExt: func(t *testing.T) *grpcClientExt {
				m := newMockAuthorizer(t)
				m.EXPECT().Context(mock.Anything).Return(ctx)
				return &grpcClientExt{oauth: m}
			},
			streamer: (&fakeInterceptor{
				stream: newMockPubSub_SubscribeClient(t),
			}).Streamer,
		},
		{
			name: "succeeds creating stream with auth error",
			grpcExt: func(t *testing.T) *grpcClientExt {
				m := newMockAuthorizer(t)
				m.EXPECT().Authorize(mock.Anything).Return(nil)
				m.EXPECT().Context(mock.Anything).Return(ctx)
				return &grpcClientExt{oauth: m}
			},
			streamer: (&fakeInterceptor{
				err:    status.Error(codes.Unauthenticated, "failed to auth"),
				tries:  1,
				stream: newMockPubSub_SubscribeClient(t),
			}).Streamer,
		},
		{
			name: "fails with error",
			grpcExt: func(t *testing.T) *grpcClientExt {
				m := newMockAuthorizer(t)
				m.EXPECT().Context(mock.Anything).Return(ctx)
				return &grpcClientExt{oauth: m}
			},
			streamer: (&fakeInterceptor{
				err: status.Error(codes.Unavailable, "failed"),
			}).Streamer,
			wantErr: status.Error(codes.Unavailable, "failed"),
		},
		{
			name: "fails with auth error",
			grpcExt: func(t *testing.T) *grpcClientExt {
				m := newMockAuthorizer(t)
				m.EXPECT().Context(mock.Anything).Return(ctx)
				m.EXPECT().Authorize(mock.Anything).Return(errors.New("boom"))
				return &grpcClientExt{oauth: m}
			},
			streamer: (&fakeInterceptor{
				err:   status.Error(codes.Unauthenticated, "failed to auth"),
				tries: 1,
			}).Streamer,
			wantErr: errors.New("failed to authorize client: boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			stream, err := tc.grpcExt(t).streamInterceptor(
				ctx,
				&grpc.StreamDesc{StreamName: "test"},
				&grpc.ClientConn{},
				"streamer",
				tc.streamer,
			)
			if tc.wantErr != nil {
				is.Equal(tc.wantErr.Error(), err.Error())
			} else {
				is.NoErr(err)
				is.True(stream != nil)
			}
		})
	}
}

func Test_unaryInterceptor(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name    string
		grpcExt func(t *testing.T) *grpcClientExt
		invoker grpc.UnaryInvoker
		wantErr error
	}{
		{
			name: "succeeds invoke",
			grpcExt: func(t *testing.T) *grpcClientExt {
				m := newMockAuthorizer(t)
				m.EXPECT().Context(mock.Anything).Return(ctx)
				return &grpcClientExt{oauth: m}
			},
			invoker: (&fakeInterceptor{}).UnaryInvoker,
		},
		{
			name: "succeeds invoke with auth error",
			grpcExt: func(t *testing.T) *grpcClientExt {
				m := newMockAuthorizer(t)
				m.EXPECT().Authorize(mock.Anything).Return(nil)
				m.EXPECT().Context(mock.Anything).Return(ctx)
				return &grpcClientExt{oauth: m}
			},
			invoker: (&fakeInterceptor{
				err:   status.Error(codes.Unauthenticated, "failed to auth"),
				tries: 1,
			}).UnaryInvoker,
		},
		{
			name: "fails with error",
			grpcExt: func(t *testing.T) *grpcClientExt {
				m := newMockAuthorizer(t)
				m.EXPECT().Context(mock.Anything).Return(ctx)
				return &grpcClientExt{oauth: m}
			},
			invoker: (&fakeInterceptor{
				err: status.Error(codes.PermissionDenied, "failed"),
			}).UnaryInvoker,
			wantErr: status.Error(codes.PermissionDenied, "failed"),
		},
		{
			name: "fails with auth error",
			grpcExt: func(t *testing.T) *grpcClientExt {
				m := newMockAuthorizer(t)
				m.EXPECT().Context(mock.Anything).Return(ctx)
				m.EXPECT().Authorize(mock.Anything).Return(errors.New("boom"))
				return &grpcClientExt{oauth: m}
			},
			invoker: (&fakeInterceptor{
				err:   status.Error(codes.Unauthenticated, "failed to auth"),
				tries: 1,
			}).UnaryInvoker,
			wantErr: errors.New("failed to authorize client: boom"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			err := tc.grpcExt(t).
				unaryInterceptor(ctx, "method", "req", "reply", &grpc.ClientConn{}, tc.invoker)

			if tc.wantErr != nil {
				is.Equal(tc.wantErr.Error(), err.Error())
			} else {
				is.NoErr(err)
			}
		})
	}
}

type fakeInterceptor struct {
	called, tries int
	stream        grpc.ClientStream
	err           error
}

func (f *fakeInterceptor) UnaryInvoker(_ context.Context, method string, req, reply any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
	if method != "method" || req != "req" || reply != "reply" {
		return errors.New("invalid args")
	}
	if f.tries > 0 && f.tries == f.called {
		return nil
	}
	f.called++
	return f.err
}

func (f *fakeInterceptor) Streamer(_ context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, method string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
	if method != "streamer" {
		return nil, errors.New("invalid args")
	}
	if f.tries > 0 && f.tries == f.called {
		return f.stream, nil
	}
	f.called++
	return f.stream, f.err
}
