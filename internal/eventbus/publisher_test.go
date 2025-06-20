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
	"strings"
	"testing"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/conduitio/conduit-commons/opencdc"
	"github.com/matryer/is"
	"github.com/stretchr/testify/mock"
)

func TestPublisher_New(t *testing.T) {
	ctx := context.TODO()

	t.Run("creates publisher", func(t *testing.T) {
		is := is.New(t)
		m := newMockClient(t)
		m.EXPECT().CanPublish(ctx, "topic1").Return(nil)

		pub, err := NewPublisher(ctx, m, "topic1")
		is.True(pub != nil)
		is.NoErr(err)
	})

	t.Run("fails to create publisher", func(t *testing.T) {
		is := is.New(t)
		m := newMockClient(t)
		m.EXPECT().CanPublish(ctx, "topic1").Return(errors.New("denied"))

		pub, err := NewPublisher(ctx, m, "topic1")
		is.True(pub == nil)
		is.Equal(err.Error(), `failed to validate topic "topic1" access: denied`)
	})
}

func TestPublisher_Publish(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name    string
		pub     func(t *testing.T) *Publisher
		r       opencdc.Record
		wantErr error
	}{
		{
			name: "success",
			pub: func(t *testing.T) *Publisher {
				is := is.New(t)
				m := newMockClient(t)

				m.EXPECT().CanPublish(ctx, "topic1").Return(nil)
				m.EXPECT().TopicSchemaID(ctx, "topic1").Return("schema-id", nil)
				m.EXPECT().Marshal(
					ctx,
					"schema-id",
					opencdc.StructuredData{"produced": "message"},
				).Return([]byte(`data`), nil)
				m.EXPECT().Publish(ctx, "topic1", mock.Anything).Return(
					&eventbusv1.PublishResponse{Results: []*eventbusv1.PublishResult{
						{ReplayId: []byte(`replay-id`)},
					}},
					nil,
				)
				pub, err := NewPublisher(ctx, m, "topic1")
				is.NoErr(err)
				return pub
			},
			r: opencdc.Record{Payload: opencdc.Change{
				After: opencdc.StructuredData{"produced": "message"},
			}},
		},
		{
			name: "schema id invalid",
			pub: func(t *testing.T) *Publisher {
				is := is.New(t)
				m := newMockClient(t)

				m.EXPECT().CanPublish(ctx, "topic1").Return(nil)
				m.EXPECT().TopicSchemaID(ctx, "topic1").Return("", errors.New("no such topic"))

				pub, err := NewPublisher(ctx, m, "topic1")
				is.NoErr(err)
				return pub
			},
			r: opencdc.Record{Payload: opencdc.Change{
				After: opencdc.StructuredData{"produced": "message"},
			}},
			wantErr: errors.New(`failed to get topic "topic1" schema: no such topic`),
		},
		{
			name: "failed to marshal record",
			pub: func(t *testing.T) *Publisher {
				is := is.New(t)
				m := newMockClient(t)

				m.EXPECT().CanPublish(ctx, "topic1").Return(nil)
				m.EXPECT().TopicSchemaID(ctx, "topic1").Return("schema-id", nil)
				m.EXPECT().Marshal(
					ctx,
					"schema-id",
					opencdc.StructuredData{"produced": "message"},
				).Return(nil, errors.New("failed to encode"))

				pub, err := NewPublisher(ctx, m, "topic1")
				is.NoErr(err)
				return pub
			},
			r: opencdc.Record{Payload: opencdc.Change{
				After: opencdc.StructuredData{"produced": "message"},
			}},
			wantErr: errors.New(`failed to encode record: failed to encode`),
		},
		{
			name: "publish fails",
			pub: func(t *testing.T) *Publisher {
				is := is.New(t)
				m := newMockClient(t)

				m.EXPECT().CanPublish(ctx, "topic1").Return(nil)
				m.EXPECT().TopicSchemaID(ctx, "topic1").Return("schema-id", nil)
				m.EXPECT().Marshal(
					ctx,
					"schema-id",
					opencdc.StructuredData{"produced": "message"},
				).Return([]byte(`data`), nil)
				m.EXPECT().Publish(ctx, "topic1", mock.Anything).Return(nil, errors.New("cant do this"))

				pub, err := NewPublisher(ctx, m, "topic1")
				is.NoErr(err)
				return pub
			},
			r: opencdc.Record{Payload: opencdc.Change{
				After: opencdc.StructuredData{"produced": "message"},
			}},
			wantErr: errors.New(`failed to send events: cant do this`),
		},
		{
			name: "publish response has err",
			pub: func(t *testing.T) *Publisher {
				is := is.New(t)
				m := newMockClient(t)

				m.EXPECT().CanPublish(ctx, "topic1").Return(nil)
				m.EXPECT().TopicSchemaID(ctx, "topic1").Return("schema-id", nil)
				m.EXPECT().Marshal(
					ctx,
					"schema-id",
					opencdc.StructuredData{"produced": "message"},
				).Return([]byte(`data`), nil)
				m.EXPECT().Publish(ctx, "topic1", mock.Anything).Return(
					&eventbusv1.PublishResponse{Results: []*eventbusv1.PublishResult{
						{
							ReplayId: []byte(`replay-id`),
							Error: &eventbusv1.Error{
								Code: eventbusv1.ErrorCode_PUBLISH,
								Msg:  "this message not good",
							},
						},
					}},
					nil,
				)

				pub, err := NewPublisher(ctx, m, "topic1")
				is.NoErr(err)
				return pub
			},
			r: opencdc.Record{Payload: opencdc.Change{
				After: opencdc.StructuredData{"produced": "message"},
			}},
			wantErr: errors.New(`failed to publish on "topic1": this message not good`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			pub := tc.pub(t)

			n, err := pub.Publish(ctx, []opencdc.Record{tc.r})
			if tc.wantErr != nil {
				is.True(err != nil)
				if !strings.Contains(err.Error(), tc.wantErr.Error()) {
					is.Equal(err.Error(), tc.wantErr.Error())
				}
			} else {
				is.Equal(n, 1)
				is.NoErr(err)
			}
		})
	}
}
