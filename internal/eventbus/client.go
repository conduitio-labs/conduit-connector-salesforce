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
	"strings"
	"time"

	"github.com/conduitio-labs/conduit-connector-salesforce/config"
	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/conduitio/conduit-commons/opencdc"
	"github.com/go-errors/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	grpcTimeout     = 5 * time.Second
	ErrEndOfRecords = errors.New("end of records from stream")
)

type schemaClient interface {
	Marshal(context.Context, string, map[string]any) ([]byte, error)
	Unmarshal(context.Context, string, []byte) (map[string]any, error)
}

type Client interface {
	subClient
	pubClient
	ParsePreset(string) (eventbusv1.ReplayPreset, error)
}

type client struct {
	config config.Config
	conn   *grpc.ClientConn
	schema schemaClient
	pubsub eventbusv1.PubSubClient
	oauth  authorizer
}

// NewClient creates a new pubsub client.
// Returns error when:
// * gRPC client fails to dial
// * Fails to auth.
func NewClient(ctx context.Context, config config.Config) (Client, error) {
	oauth, err := newAuthorizer(config.ClientID, config.ClientSecret, config.OAuthEndpoint)
	if err != nil {
		return nil, errors.Errorf("failed to initialize auth: %w", err)
	}

	if err := oauth.Authorize(ctx); err != nil {
		return nil, errors.Errorf("failed to authorize credentials: %w", err)
	}

	conn, err := grpc.NewClient(
		config.PubsubAddress,
		newClientExt(config, oauth).DialOpts()...,
	)
	if err != nil {
		return nil, errors.Errorf("gRPC dial: %w", err)
	}

	pubsub := eventbusv1.NewPubSubClient(conn)

	return &client{
		config: config,
		schema: newSchemaClient(pubsub),
		conn:   conn,
		pubsub: pubsub,
		oauth:  oauth,
	}, nil
}

// Teardown closes the underlying eventbus gRPC connections.
func (c *client) Teardown(_ context.Context) error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Authorize refreshes the oauth and returns an auth context.
// Returns error when auth fails.
func (c *client) Authorize(ctx context.Context) (context.Context, error) {
	if err := c.oauth.Authorize(ctx); err != nil {
		return ctx, errors.Errorf("failed to authorize: %w", err)
	}

	return c.oauth.Context(ctx), nil
}

// Marshal serializes the provided structured data using avro with the schema from the topic.
// Returns error when serialization fails.
func (c *client) Marshal(ctx context.Context, schemaID string, data opencdc.StructuredData) ([]byte, error) {
	v, err := c.schema.Marshal(ctx, schemaID, data)
	if err != nil {
		return nil, errors.Errorf("failed to marshal data with schema %q: %w", schemaID, err)
	}

	return v, nil
}

// Marshal serializes the provided structured data using avro with the schema from the topic.
// Returns error when serialization fails.
func (c *client) Unmarshal(ctx context.Context, schemaID string, v []byte) (opencdc.StructuredData, error) {
	data, err := c.schema.Unmarshal(ctx, schemaID, v)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal data with schema %q: %w", schemaID, err)
	}

	return opencdc.StructuredData(data), nil
}

// TopicSchemaID returns the schema ID for the specific topic.
// Schema ID can be used to marshal/unmarshal data in and out of avro format.
// Returns error on failure to describe topic.
func (c *client) TopicSchemaID(ctx context.Context, topic string) (string, error) {
	t, err := c.pubsub.GetTopic(ctx, &eventbusv1.TopicRequest{
		TopicName: topic,
	})
	if err != nil {
		return "", errors.Errorf("failed to describe topic %q: %w", topic, err)
	}

	// retry on auth
	return t.SchemaId, nil
}

// Subscribe creates streaming client which will be used to send requests and receive events.
func (c *client) Subscribe(ctx context.Context) (eventbusv1.PubSub_SubscribeClient, error) {
	sub, err := c.pubsub.Subscribe(ctx)
	if err != nil {
		return nil, errors.Errorf("failed to subscribe: %w", err)
	}
	return sub, nil
}

// Publish produces the provided events to the eventbus topic. Failure to produce will be retried with
// fresh auth token and validated topic permissions.
// Returns error when:
// * On publish failure, token cannot be obtained (authorization errors)
// * Client is not allowed to publish to the provided topic.
// * Eventbus returned an error on produce.
func (c *client) Publish(ctx context.Context, topic string, ee []*eventbusv1.ProducerEvent) (*eventbusv1.PublishResponse, error) {
	var (
		resp       *eventbusv1.PublishResponse
		publishErr error
	)

	req := &eventbusv1.PublishRequest{
		TopicName: topic,
		Events:    ee,
	}

	resp, publishErr = c.pubsub.Publish(ctx, req)
	if publishErr == nil {
		return resp, nil
	}

	if err := c.CanPublish(ctx, topic); err != nil {
		return nil, errors.Errorf("failed to validate client %q access to %q: %w", c.config.ClientID, topic, err)
	}

	resp, publishErr = c.pubsub.Publish(ctx, req)
	if publishErr != nil {
		return nil, errors.Errorf("failed to publish to %q: %w", topic, publishErr)
	}

	return resp, nil
}

// CanPublish returns an error when the client is not allowed to publish events
// to the list of topics.
func (c *client) CanPublish(ctx context.Context, topics ...string) error {
	return c.checkTopic(ctx, func(t *eventbusv1.TopicInfo) error {
		if !t.CanPublish {
			return errors.Errorf("not allowed to publish to %q", t.TopicName)
		}
		return nil
	}, topics...)
}

// CanSubscribe returns an error when the client is not allowed to subscribe to
// events on the list of topics.
func (c *client) CanSubscribe(ctx context.Context, topics ...string) error {
	return c.checkTopic(ctx, func(t *eventbusv1.TopicInfo) error {
		if !t.CanSubscribe {
			return errors.Errorf("not allowed to subscribe to %q", t.TopicName)
		}
		return nil
	}, topics...)
}

// ParsePreset converts the string to the proto type `ReplayPreset`.
func (*client) ParsePreset(preset string) (eventbusv1.ReplayPreset, error) {
	v, ok := eventbusv1.ReplayPreset_value[strings.ToUpper(preset)]
	if !ok {
		return eventbusv1.ReplayPreset(-1), errors.Errorf("invalid preset %q", preset)
	}
	return eventbusv1.ReplayPreset(v), nil
}

// checkTopicFn defines a function which evaluates the topic metadata.
// Should return an error when check fails.
type checkTopicFn func(*eventbusv1.TopicInfo) error

// checkTopic will run the provided check function over the metadata of each topic.
// Returns error accumulated from each topic's check.
func (c *client) checkTopic(ctx context.Context, fn checkTopicFn, topics ...string) error {
	var errs error

	for _, topic := range topics {
		resp, err := c.getTopic(ctx, topic)
		if err != nil {
			return errors.Errorf("failed to fetch topic %q: %w", topic, err)
		}

		if err := fn(resp); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

// getTopic returns eventbus information about the specific topic.
// Returns error when unable to fetch topic info.
func (c *client) getTopic(ctx context.Context, topic string) (*eventbusv1.TopicInfo, error) {
	var trailer metadata.MD

	req := &eventbusv1.TopicRequest{
		TopicName: topic,
	}

	ctx, cancel := context.WithTimeout(ctx, grpcTimeout)
	defer cancel()

	resp, err := c.pubsub.GetTopic(ctx, req, grpc.Trailer(&trailer))
	if err != nil {
		return nil, errors.Errorf("failed to retrieve topic %q: %w", topic, err)
	}
	return resp, nil
}
