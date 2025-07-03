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

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
	"github.com/google/uuid"
)

type pubClient interface {
	CanPublish(context.Context, ...string) error
	Publish(context.Context, string, []*eventbusv1.ProducerEvent) (*eventbusv1.PublishResponse, error)
	Marshal(context.Context, string, opencdc.StructuredData) ([]byte, error)
	TopicSchemaID(context.Context, string) (string, error)
	Teardown(context.Context) error
}

// Publisher is produce-only client to the eventbus service.
type Publisher struct {
	c     pubClient
	topic string
}

// NewPublisher creates new eventbus publisher for the specific topic.
// Returns error when:
// * Failed to initialize the eventbus client.
// * Publishing to the specific topic is disallowed.
func NewPublisher(ctx context.Context, c pubClient, topic string) (*Publisher, error) {
	if err := c.CanPublish(ctx, topic); err != nil {
		return nil, errors.Errorf("failed to validate topic %q access: %w", topic, err)
	}

	return &Publisher{
		c:     c,
		topic: topic,
	}, nil
}

// Teardown will close the underlying client connection.
func (p *Publisher) Teardown(ctx context.Context) error {
	if p.c != nil {
		return p.c.Teardown(ctx)
	}
	return nil
}

// Publish attempts to write all records to Eventbus. Any failed records will be retried
// until failure to publish at least one using partial writes.
// Returns the number of published records and error on failure.
func (p *Publisher) Publish(ctx context.Context, rr []opencdc.Record) (int, error) {
	var n int

	for n != len(rr) {
		published, err := p.publishEvents(ctx, rr[n:])
		if err != nil && published == 0 {
			return n, errors.Errorf("failed to publish events: %w", err)
		}
		n += published
	}

	return n, nil
}

// publishEvents will encode all records with avro using the schema id of the topic.
// Partial writes are possible. Returns the number of written records and error on failure.
func (p *Publisher) publishEvents(ctx context.Context, rr []opencdc.Record) (int, error) {
	events, err := p.marshalEvents(ctx, rr)
	if err != nil {
		return 0, errors.Errorf("failed to encode records: %w", err)
	}

	resp, err := p.c.Publish(ctx, p.topic, events)
	if err != nil {
		return 0, errors.Errorf("failed to send events: %w", err)
	}

	// assert equivalent number of responses have been returned
	if len(resp.Results) != len(rr) {
		return 0, errors.Errorf("fatal error got %d out of %d", len(resp.Results), len(rr))
	}

	for i, result := range resp.Results {
		if err := result.GetError(); err != nil {
			sdk.Logger(ctx).Error().Interface("error", err).
				Str("correlation_key", result.CorrelationKey).
				Str("topic", p.topic).
				Msg("failed to publish")
			return i, errors.Errorf("failed to publish on %q: %s", p.topic, err.Msg)
		}
	}

	return len(rr), nil
}

// marshalEvents creates eventbus events by encoding the `.Payload.After` of the OpenCDC record
// using avro with the associated topic schema. Returns a slice of producer events.
// Returns error when avro serialization fails.
func (p *Publisher) marshalEvents(ctx context.Context, rr []opencdc.Record) ([]*eventbusv1.ProducerEvent, error) {
	encoded := make([]*eventbusv1.ProducerEvent, 0, len(rr))

	schemaID, err := p.c.TopicSchemaID(ctx, p.topic)
	if err != nil {
		return nil, errors.Errorf("failed to get topic %q schema: %w", p.topic, err)
	}

	for _, r := range rr {
		data, err := p.c.Marshal(ctx, schemaID, r.Payload.After.(opencdc.StructuredData))
		if err != nil {
			return encoded, errors.Errorf("failed to encode record: %w", err)
		}

		encoded = append(encoded, &eventbusv1.ProducerEvent{
			SchemaId: schemaID,
			Payload:  data,
			Id:       uuid.NewString(),
		})
	}

	return encoded, nil
}
