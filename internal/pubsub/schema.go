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

package pubsub

import (
	"context"
	"sync"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/go-errors/errors"
	"github.com/hamba/avro"
)

type Schema struct {
	mu    sync.Mutex
	c     eventbusv1.PubSubClient
	cache map[string]avro.Schema
}

func newSchema(c eventbusv1.PubSubClient) *Schema {
	return &Schema{
		c:     c,
		cache: make(map[string]avro.Schema),
		mu:    sync.Mutex{},
	}
}

func (s *Schema) Unmarshal(ctx context.Context, schemaID string, v []byte) (map[string]any, error) {
	schema, err := s.schema(ctx, schemaID)
	if err != nil {
		return nil, errors.Errorf("failed to retrieve schema %q: %w", schemaID, err)
	}

	data := make(map[string]any)
	if err := avro.Unmarshal(schema, v, &data); err != nil {
		return nil, errors.Errorf("failed to unmarshal with schema %q: %w", schemaID, err)
	}

	return data, nil
}

func (s *Schema) Marshal(ctx context.Context, schemaID string, data map[string]any) ([]byte, error) {
	schema, err := s.schema(ctx, schemaID)
	if err != nil {
		return nil, errors.Errorf("failed to retrieve schema %q: %w", schemaID, err)
	}

	v, err := avro.Marshal(schema, data)
	if err != nil {
		return nil, errors.Errorf("failed to marshal with schema %q: %w", schemaID, err)
	}

	return v, nil
}

func (s *Schema) schema(ctx context.Context, schemaID string) (avro.Schema, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s, ok := s.cache[schemaID]; ok {
		return s, nil
	}

	resp, err := s.c.GetSchema(ctx, &eventbusv1.SchemaRequest{
		SchemaId: schemaID,
	})
	if err != nil {
		return nil, errors.Errorf("failed to retrieve schema %q: %w", schemaID, err)
	}

	parsed, err := avro.Parse(resp.GetSchemaJson())
	if err != nil {
		return nil, errors.Errorf("failed to parse schema %q: %w", schemaID, err)
	}
	s.cache[schemaID] = parsed

	return parsed, nil
}
