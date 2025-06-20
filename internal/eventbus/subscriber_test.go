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
	"io"
	"strings"
	"testing"
	"time"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/conduitio/conduit-commons/opencdc"
	"github.com/matryer/is"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSubscriber_New(t *testing.T) {
	ctx := context.TODO()

	t.Run("creates subscriber", func(t *testing.T) {
		is := is.New(t)
		ch := make(chan *EventData)

		m := newMockClient(t)
		m.EXPECT().CanSubscribe(ctx, "topic1").Return(nil)
		m.EXPECT().Subscribe(ctx).Return(newMockPubSub_SubscribeClient(t), nil)

		s, err := NewSubscriber(ctx, m, ch, TopicConfig{
			TopicName:    "topic1",
			ReplayID:     []byte("foobar"),
			ReplayPreset: eventbusv1.ReplayPreset_EARLIEST,
		})
		is.NoErr(err)
		is.True(s != nil)
	})

	t.Run("no access to topic", func(t *testing.T) {
		is := is.New(t)
		ch := make(chan *EventData)

		m := newMockClient(t)
		m.EXPECT().CanSubscribe(ctx, "topic1").Return(errors.New("cant touch this"))

		_, err := NewSubscriber(ctx, m, ch, TopicConfig{
			TopicName:    "topic1",
			ReplayID:     []byte("foobar"),
			ReplayPreset: eventbusv1.ReplayPreset_EARLIEST,
		})
		is.True(err != nil)
		is.Equal(err.Error(), `unable to subscribe to topic "topic1": cant touch this`)
	})

	t.Run("fails to subscribe", func(t *testing.T) {
		is := is.New(t)
		ch := make(chan *EventData)

		m := newMockClient(t)
		m.EXPECT().CanSubscribe(ctx, "topic1").Return(nil)
		m.EXPECT().Subscribe(ctx).Return(nil, errors.New("bad error"))

		_, err := NewSubscriber(ctx, m, ch, TopicConfig{
			TopicName:    "topic1",
			ReplayID:     []byte("foobar"),
			ReplayPreset: eventbusv1.ReplayPreset_EARLIEST,
		})
		is.True(err != nil)
		is.Equal(err.Error(), `failed to subscribe to "topic1": bad error`)
	})
}

func TestSubscriber_Run(t *testing.T) {
	tests := []struct {
		name        string
		sub         func(t *testing.T, ctx context.Context) *Subscriber
		expectEvent *EventData
		wantErr     error
	}{
		{
			name: "success",
			sub: func(t *testing.T, ctx context.Context) *Subscriber {
				t.Helper()
				is := is.New(t)

				req, resp := testSendRecv(t)
				mockStreamClient := newMockPubSub_SubscribeClient(t)
				mockStreamClient.EXPECT().Send(req).Return(nil)
				mockStreamClient.EXPECT().Recv().Return(resp, nil).Once()
				mockStreamClient.EXPECT().Recv().RunAndReturn(func() (*eventbusv1.FetchResponse, error) {
					<-ctx.Done()
					return &eventbusv1.FetchResponse{}, nil
				})

				mockSubClient := newMockClient(t)
				mockSubClient.EXPECT().CanSubscribe(ctx, "topic1").Return(nil)
				mockSubClient.EXPECT().Subscribe(ctx).Return(mockStreamClient, nil)
				mockSubClient.EXPECT().Unmarshal(ctx, "schema-test", []byte(`data`)).
					Return(opencdc.StructuredData{"message": "expected"}, nil)

				s, err := NewSubscriber(ctx, mockSubClient, make(chan *EventData), TopicConfig{
					TopicName:    "topic1",
					ReplayID:     []byte("position"),
					ReplayPreset: eventbusv1.ReplayPreset_EARLIEST,
				})
				is.NoErr(err)
				return s
			},
			expectEvent: &EventData{
				ID:       "test-id",
				Topic:    "topic1",
				ReplayID: []byte(`replay-id-1`),
				Data:     opencdc.StructuredData{"message": "expected"},
			},
			wantErr: errors.New(`context canceled`),
		},
		{
			name: "recv fails",
			sub: func(t *testing.T, ctx context.Context) *Subscriber {
				t.Helper()
				is := is.New(t)

				req, _ := testSendRecv(t)
				mockStreamClient := newMockPubSub_SubscribeClient(t)
				mockStreamClient.EXPECT().Send(req).Return(io.EOF).Once() // no failure here
				mockStreamClient.EXPECT().Recv().Return(nil, errors.New("recv failed")).Once()

				mockSubClient := newMockClient(t)
				mockSubClient.EXPECT().CanSubscribe(ctx, "topic1").Return(nil)
				mockSubClient.EXPECT().Subscribe(ctx).Return(mockStreamClient, nil)

				s, err := NewSubscriber(ctx, mockSubClient, make(chan *EventData), TopicConfig{
					TopicName:    "topic1",
					ReplayID:     []byte("position"),
					ReplayPreset: eventbusv1.ReplayPreset_EARLIEST,
				})
				is.NoErr(err)
				return s
			},
			wantErr: errors.New(`failed to receive from "topic1" stream: recv failed`),
		},
		{
			name: "send failed",
			sub: func(t *testing.T, ctx context.Context) *Subscriber {
				t.Helper()
				is := is.New(t)

				req, _ := testSendRecv(t)
				mockStreamClient := newMockPubSub_SubscribeClient(t)
				mockStreamClient.EXPECT().Send(req).Return(errors.New("send failed")).Once()

				mockSubClient := newMockClient(t)
				mockSubClient.EXPECT().CanSubscribe(ctx, "topic1").Return(nil)
				mockSubClient.EXPECT().Subscribe(ctx).Return(mockStreamClient, nil)

				s, err := NewSubscriber(ctx, mockSubClient, make(chan *EventData), TopicConfig{
					TopicName:    "topic1",
					ReplayID:     []byte("position"),
					ReplayPreset: eventbusv1.ReplayPreset_EARLIEST,
				})
				is.NoErr(err)
				return s
			},
			wantErr: errors.New(`failure to send fetch request: send failed`),
		},
		{
			name: "failed to build event",
			sub: func(t *testing.T, ctx context.Context) *Subscriber {
				t.Helper()
				is := is.New(t)

				req, resp := testSendRecv(t)
				mockStreamClient := newMockPubSub_SubscribeClient(t)
				mockStreamClient.EXPECT().Send(req).Return(nil)
				mockStreamClient.EXPECT().Recv().Return(resp, nil).Once()
				mockStreamClient.EXPECT().Recv().RunAndReturn(func() (*eventbusv1.FetchResponse, error) {
					<-ctx.Done()
					return &eventbusv1.FetchResponse{}, nil
				})

				mockSubClient := newMockClient(t)
				mockSubClient.EXPECT().CanSubscribe(ctx, "topic1").Return(nil)
				mockSubClient.EXPECT().Subscribe(ctx).Return(mockStreamClient, nil)
				mockSubClient.EXPECT().Unmarshal(ctx, "schema-test", []byte(`data`)).
					Return(nil, errors.New("failed to make event"))

				s, err := NewSubscriber(ctx, mockSubClient, make(chan *EventData), TopicConfig{
					TopicName:    "topic1",
					ReplayID:     []byte("position"),
					ReplayPreset: eventbusv1.ReplayPreset_EARLIEST,
				})
				is.NoErr(err)
				return s
			},
			wantErr: errors.New(`failed to unmarshal record: failed to make event`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			done := make(chan struct{})
			ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
			defer cancel()

			sub := tc.sub(t, ctx)

			go func() {
				defer close(done)
				if err := sub.Run(ctx); err != nil {
					is.True(strings.Contains(err.Error(), tc.wantErr.Error()))
				}
			}()

			if tc.expectEvent != nil {
				select {
				case e := <-sub.events:
					is.Equal(e, tc.expectEvent)
					cancel()
				case <-ctx.Done():
					is.Fail()
				}
			}
			<-done
		})
	}
}

func TestSubscriber_recoverErr(t *testing.T) {
	t.Run("nothing to recover", func(t *testing.T) {
		is := is.New(t)
		is.NoErr((&Subscriber{}).recoverErr(context.TODO(), nil))
	})

	t.Run("unrecoverable error", func(t *testing.T) {
		is := is.New(t)
		badErr := errors.New("bad error")

		err := (&Subscriber{}).recoverErr(context.TODO(), badErr)
		is.True(err != nil)
		is.True(errors.Is(err, badErr))
	})

	t.Run("fails to recover auth", func(t *testing.T) {
		is := is.New(t)
		badErr := errors.New("bad error")
		ctx := context.TODO()

		m := newMockClient(t)
		m.EXPECT().Authorize(ctx).Return(ctx, badErr)

		s := &Subscriber{c: m}
		err := s.recoverErr(ctx, status.Error(codes.Unauthenticated, "auth"))
		is.True(err != nil)
		is.Equal(err.Error(), "failed to recover auth: bad error")
	})

	t.Run("fails to re-sub after auth", func(t *testing.T) {
		is := is.New(t)
		badErr := errors.New("bad error")
		ctx := context.TODO()

		m := newMockClient(t)
		m.EXPECT().Authorize(ctx).Return(ctx, nil)
		m.EXPECT().Subscribe(ctx).Return(nil, badErr)

		s := &Subscriber{c: m}
		err := s.recoverErr(ctx, status.Error(codes.Unauthenticated, "auth"))
		is.True(err != nil)
		is.Equal(err.Error(), "failed to recover stream with fresh auth: bad error")
	})

	t.Run("recovers unauthenticated and unavailable", func(t *testing.T) {
		is := is.New(t)
		ctx := context.TODO()

		m := newMockClient(t)
		m.EXPECT().Authorize(ctx).Return(ctx, nil)
		m.EXPECT().Subscribe(ctx).Return(nil, nil)

		s := &Subscriber{c: m}
		is.NoErr(s.recoverErr(ctx, status.Error(codes.Unauthenticated, "auth")))
		is.NoErr(s.recoverErr(ctx, status.Error(codes.Unavailable, "unvail")))
	})

	t.Run("recovers invalid arg", func(t *testing.T) {
		is := is.New(t)
		ctx := context.TODO()

		m := newMockClient(t)
		m.EXPECT().Authorize(ctx).Return(ctx, nil)
		m.EXPECT().Subscribe(ctx).Return(nil, nil)

		s := &Subscriber{
			c:            m,
			replayPreset: eventbusv1.ReplayPreset_CUSTOM,
			lastReplayID: []byte("position"),
		}
		is.NoErr(s.recoverErr(ctx, status.Error(codes.InvalidArgument, "auth")))
		is.Equal(s.replayPreset, eventbusv1.ReplayPreset_EARLIEST)
		is.Equal(s.lastReplayID, nil)
	})
}

func testSendRecv(t *testing.T) (*eventbusv1.FetchRequest, *eventbusv1.FetchResponse) {
	t.Helper()

	req := &eventbusv1.FetchRequest{
		TopicName:    "topic1",
		ReplayPreset: eventbusv1.ReplayPreset_CUSTOM,
		ReplayId:     []byte("position"),
		NumRequested: 10,
	}

	resp := &eventbusv1.FetchResponse{
		Events: []*eventbusv1.ConsumerEvent{{
			Event: &eventbusv1.ProducerEvent{
				Id:       "test-id",
				SchemaId: "schema-test",
				Payload:  []byte(`data`),
			},
			ReplayId: []byte(`replay-id-1`),
		}},
		LatestReplayId:      []byte("position"),
		PendingNumRequested: 10,
	}

	return req, resp
}
