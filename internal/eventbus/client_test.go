// Copyright © 2024 Meroxa, Inc.
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
	"strings"
	"testing"

	"github.com/conduitio-labs/conduit-connector-salesforce/config"
	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/conduitio/conduit-commons/opencdc"
	"github.com/go-errors/errors"
	"github.com/matryer/is"
	"github.com/stretchr/testify/mock"
)

func TestClient_CanPublishSubscribe(t *testing.T) {
	ctx := context.TODO()

	t.Run("allowed", func(t *testing.T) {
		is := is.New(t)

		m := newMockPubSubClient(t)
		m.EXPECT().GetTopic(mock.Anything, &eventbusv1.TopicRequest{
			TopicName: "topic1",
		}, mock.Anything).Return(&eventbusv1.TopicInfo{
			CanPublish:   true,
			CanSubscribe: true,
			TopicName:    "topic1",
		}, nil)

		c := client{pubsub: m}
		is.NoErr(c.CanPublish(ctx, "topic1"))
		is.NoErr(c.CanSubscribe(ctx, "topic1"))
	})

	t.Run("disallowed", func(t *testing.T) {
		is := is.New(t)
		expectPubErr := errors.New(`not allowed to publish to "topic1"`)
		expectSubErr := errors.New(`not allowed to subscribe to "topic1"`)

		m := newMockPubSubClient(t)
		m.EXPECT().GetTopic(mock.Anything, &eventbusv1.TopicRequest{
			TopicName: "topic1",
		}, mock.Anything).Return(&eventbusv1.TopicInfo{
			CanPublish:   false,
			CanSubscribe: false,
			TopicName:    "topic1",
		}, nil)

		c := client{pubsub: m}
		pubErr := c.CanPublish(ctx, "topic1")
		if !strings.Contains(pubErr.Error(), expectPubErr.Error()) {
			is.Equal(pubErr.Error(), expectPubErr.Error())
		}

		subErr := c.CanSubscribe(ctx, "topic1")
		if !strings.Contains(subErr.Error(), expectSubErr.Error()) {
			is.Equal(subErr.Error(), expectSubErr.Error())
		}
	})

	t.Run("failed to get topic", func(t *testing.T) {
		is := is.New(t)
		expectErr := errors.New(`failed to retrieve topic "topic1": bad topic`)

		m := newMockPubSubClient(t)
		m.EXPECT().GetTopic(mock.Anything, &eventbusv1.TopicRequest{
			TopicName: "topic1",
		}, mock.Anything).Return(nil, errors.New("bad topic"))

		c := client{pubsub: m}
		pubErr := c.CanPublish(ctx, "topic1")
		if !strings.Contains(pubErr.Error(), expectErr.Error()) {
			is.Equal(pubErr.Error(), expectErr.Error())
		}

		subErr := c.CanSubscribe(ctx, "topic1")
		if !strings.Contains(subErr.Error(), expectErr.Error()) {
			is.Equal(subErr.Error(), expectErr.Error())
		}
	})
}

func TestClient_ParsePreset(t *testing.T) {
	c := client{}

	tests := []struct {
		name         string
		expectPreset eventbusv1.ReplayPreset
		wantErr      error
	}{
		{name: "earliest", expectPreset: eventbusv1.ReplayPreset_EARLIEST},
		{name: "latest", expectPreset: eventbusv1.ReplayPreset_LATEST},
		{name: "custom", expectPreset: eventbusv1.ReplayPreset_CUSTOM},
		{name: "incorrect", wantErr: errors.New(`invalid preset "incorrect"`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			parsed, err := c.ParsePreset(tc.name)
			if tc.wantErr != nil {
				is.True(err != nil)
				is.Equal(err.Error(), tc.wantErr.Error())
			} else {
				is.NoErr(err)
				is.Equal(parsed, tc.expectPreset)
			}
		})
	}
}

func TestClient_Subscribe(t *testing.T) {
	is := is.New(t)
	ctx := context.TODO()
	m := newMockPubSubClient(t)
	c := client{pubsub: m}

	m.EXPECT().Subscribe(ctx).Return(newMockPubSub_SubscribeClient(t), nil).Once()
	_, err := c.Subscribe(ctx)
	is.NoErr(err)

	m.EXPECT().Subscribe(ctx).Return(nil, errors.New("cannot subscribe")).Once()
	_, err = c.Subscribe(ctx)
	is.True(err != nil)
	is.Equal(err.Error(), "failed to subscribe: cannot subscribe")
}

func TestClient_TopicSchemaID(t *testing.T) {
	is := is.New(t)
	ctx := context.TODO()
	m := newMockPubSubClient(t)
	c := client{pubsub: m}

	m.EXPECT().GetTopic(ctx, &eventbusv1.TopicRequest{TopicName: "topic123"}).Return(
		&eventbusv1.TopicInfo{SchemaId: "schema-123", TopicName: "topic123"}, nil,
	).Once()
	schemaID, err := c.TopicSchemaID(ctx, "topic123")
	is.NoErr(err)
	is.Equal(schemaID, "schema-123")

	m.EXPECT().GetTopic(ctx, &eventbusv1.TopicRequest{TopicName: "topic4"}).Return(
		nil, errors.New("bad topic"),
	).Once()
	_, err = c.TopicSchemaID(ctx, "topic4")
	is.Equal(err.Error(), `failed to describe topic "topic4": bad topic`)
}

func TestClient_MarshalUnmarshal(t *testing.T) {
	ctx := context.TODO()
	data := opencdc.StructuredData{"some": "data"}

	t.Run("marshals data", func(t *testing.T) {
		is := is.New(t)
		m := newMockSchemaClient(t)
		c := client{schema: m}

		m.EXPECT().Marshal(ctx, "schema-123", map[string]any(data)).
			Return([]byte(`encoded-data`), nil).Once()

		v, err := c.Marshal(ctx, "schema-123", data)
		is.NoErr(err)
		is.Equal([]byte(`encoded-data`), v)
	})

	t.Run("error while marshaling", func(t *testing.T) {
		is := is.New(t)
		m := newMockSchemaClient(t)
		c := client{schema: m}

		m.EXPECT().Marshal(ctx, "schema-123", map[string]any(data)).
			Return(nil, errors.New("bad ser")).Once()

		_, err := c.Marshal(ctx, "schema-123", data)
		is.True(err != nil)
		is.Equal(err.Error(), `failed to marshal data with schema "schema-123": bad ser`)
	})

	t.Run("unmarshals data", func(t *testing.T) {
		is := is.New(t)
		m := newMockSchemaClient(t)
		c := client{schema: m}

		m.EXPECT().Unmarshal(ctx, "schema-123", []byte(`encoded-data`)).
			Return(data, nil).Once()

		v, err := c.Unmarshal(ctx, "schema-123", []byte(`encoded-data`))
		is.NoErr(err)
		is.Equal(data, v)
	})

	t.Run("error while unmarshaling", func(t *testing.T) {
		is := is.New(t)
		m := newMockSchemaClient(t)
		c := client{schema: m}

		m.EXPECT().Unmarshal(ctx, "schema-123", []byte(`encoded-data`)).
			Return(nil, errors.New("bad der")).Once()

		_, err := c.Unmarshal(ctx, "schema-123", []byte(`encoded-data`))
		is.True(err != nil)
		is.Equal(err.Error(), `failed to unmarshal data with schema "schema-123": bad der`)
	})
}

func TestClient_Authorize(t *testing.T) {
	is := is.New(t)
	m := newMockAuthorizer(t)
	c := client{oauth: m}
	ctx := context.TODO()
	authCtx := context.TODO()

	m.EXPECT().Authorize(ctx).Return(nil).Once()
	m.EXPECT().Context(ctx).Return(authCtx)
	actx, err := c.Authorize(ctx)
	is.NoErr(err)
	is.Equal(authCtx, actx)

	m.EXPECT().Authorize(ctx).Return(errors.New("denied")).Once()
	_, err = c.Authorize(ctx)
	is.True(err != nil)
	is.Equal(err.Error(), "failed to authorize: denied")
}

func TestClient_Publish(t *testing.T) {
	ctx := context.TODO()

	tests := []struct {
		name           string
		client         func(t *testing.T) *client
		expectResponse *eventbusv1.PublishResponse
		wantErr        error
	}{
		{
			name: "success",
			client: func(t *testing.T) *client {
				m := newMockPubSubClient(t)

				req := &eventbusv1.PublishRequest{
					TopicName: "topic1",
					Events: []*eventbusv1.ProducerEvent{
						{Id: "event-id", SchemaId: "schema-123", Payload: []byte(`encoded-data`)},
					},
				}

				m.EXPECT().Publish(ctx, req).Return(&eventbusv1.PublishResponse{
					Results: []*eventbusv1.PublishResult{
						{ReplayId: []byte(`replay-id`)},
					},
					SchemaId: "schema-123",
				}, nil).Once()

				return &client{pubsub: m}
			},
			expectResponse: &eventbusv1.PublishResponse{
				Results: []*eventbusv1.PublishResult{
					{ReplayId: []byte(`replay-id`)},
				},
				SchemaId: "schema-123",
			},
		},
		{
			name: "success with retry",
			client: func(t *testing.T) *client {
				m := newMockPubSubClient(t)
				req := &eventbusv1.PublishRequest{
					TopicName: "topic1",
					Events: []*eventbusv1.ProducerEvent{
						{Id: "event-id", SchemaId: "schema-123", Payload: []byte(`encoded-data`)},
					},
				}
				m.EXPECT().Publish(ctx, req).Return(nil, errors.New("bad publish")).Once()
				m.EXPECT().GetTopic(
					mock.Anything,
					&eventbusv1.TopicRequest{TopicName: "topic1"},
					mock.Anything,
				).Return(&eventbusv1.TopicInfo{CanPublish: true}, nil).Once()

				m.EXPECT().Publish(ctx, req).Return(&eventbusv1.PublishResponse{
					Results: []*eventbusv1.PublishResult{
						{ReplayId: []byte(`replay-id`)},
					},
					SchemaId: "schema-123",
				}, nil).Once()

				return &client{pubsub: m}
			},
			expectResponse: &eventbusv1.PublishResponse{
				Results: []*eventbusv1.PublishResult{
					{ReplayId: []byte(`replay-id`)},
				},
				SchemaId: "schema-123",
			},
		},
		{
			name: "fails with retry",
			client: func(t *testing.T) *client {
				m := newMockPubSubClient(t)
				req := &eventbusv1.PublishRequest{
					TopicName: "topic1",
					Events: []*eventbusv1.ProducerEvent{
						{Id: "event-id", SchemaId: "schema-123", Payload: []byte(`encoded-data`)},
					},
				}
				m.EXPECT().Publish(ctx, req).Return(nil, errors.New("bad publish")).Once()
				m.EXPECT().GetTopic(
					mock.Anything,
					&eventbusv1.TopicRequest{TopicName: "topic1"},
					mock.Anything,
				).Return(&eventbusv1.TopicInfo{TopicName: "topic1", CanPublish: false}, nil).Once()

				return &client{pubsub: m, config: config.Config{ClientID: "client-id"}}
			},
			wantErr: errors.New(`failed to validate client "client-id" access to "topic1": not allowed to publish to "topic1"`),
		},
		{
			name: "fails after retry",
			client: func(t *testing.T) *client {
				m := newMockPubSubClient(t)
				req := &eventbusv1.PublishRequest{
					TopicName: "topic1",
					Events: []*eventbusv1.ProducerEvent{
						{Id: "event-id", SchemaId: "schema-123", Payload: []byte(`encoded-data`)},
					},
				}
				m.EXPECT().Publish(ctx, req).Return(nil, errors.New("bad publish")).Once()
				m.EXPECT().GetTopic(
					mock.Anything,
					&eventbusv1.TopicRequest{TopicName: "topic1"},
					mock.Anything,
				).Return(&eventbusv1.TopicInfo{CanPublish: true}, nil).Once()

				m.EXPECT().Publish(ctx, req).Return(nil, errors.New("real bad publish")).Once()

				return &client{pubsub: m}
			},
			wantErr: errors.New(`failed to publish to "topic1": real bad publish`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			c := tc.client(t)

			resp, err := c.Publish(ctx, "topic1", []*eventbusv1.ProducerEvent{{
				Id:       "event-id",
				SchemaId: "schema-123",
				Payload:  []byte(`encoded-data`),
			}},
			)
			if tc.wantErr != nil {
				is.True(err != nil)
				is.Equal(err.Error(), tc.wantErr.Error())
			} else {
				is.NoErr(err)
				is.Equal(resp, tc.expectResponse)
			}
		})
	}
}
