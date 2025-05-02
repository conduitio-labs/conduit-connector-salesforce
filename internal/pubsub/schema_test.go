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
	_ "embed"
	"errors"
	"testing"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/hamba/avro"
	"github.com/matryer/is"
)

//go:embed testdata/avro.schema
var testSchema string

func Test_SchemaUnmarshal(t *testing.T) {
	ctx := context.Background()
	decoded, encoded := encodedTestData(t, testSchema)

	tests := []struct {
		name         string
		schemaClient func(t *testing.T) *SchemaClient
		schemaID     string
		data         []byte
		expected     map[string]any
		wantErr      error
	}{
		{
			name:     "unmarshal success",
			schemaID: "my-schema-123",
			schemaClient: func(t *testing.T) *SchemaClient {
				t.Helper()

				c := newMockPubSubClient(t)
				c.EXPECT().GetSchema(ctx, &eventbusv1.SchemaRequest{
					SchemaId: "my-schema-123",
				}).Return(&eventbusv1.SchemaInfo{
					SchemaId:   "my-schema-123",
					SchemaJson: testSchema,
				}, nil)

				return newSchemaClient(c)
			},
			data:     encoded,
			expected: decoded,
		},
		{
			name:     "cached unmarshal success ",
			schemaID: "my-schema-123",
			schemaClient: func(t *testing.T) *SchemaClient {
				t.Helper()
				is := is.New(t)

				c := newMockPubSubClient(t)
				sch := newSchemaClient(c)
				parsed, err := avro.Parse(testSchema)
				is.NoErr(err)
				sch.cache["my-schema-123"] = parsed
				return sch
			},
			data:     encoded,
			expected: decoded,
		},
		{
			name:     "fail to get schema",
			schemaID: "my-schema-123",
			schemaClient: func(t *testing.T) *SchemaClient {
				t.Helper()

				c := newMockPubSubClient(t)
				c.EXPECT().GetSchema(ctx, &eventbusv1.SchemaRequest{
					SchemaId: "my-schema-123",
				}).Return(nil, errors.New("boom"))
				return newSchemaClient(c)
			},
			wantErr: errors.New(`failed to retrieve schema "my-schema-123": boom`),
		},
		{
			name:     "fail to parse schema",
			schemaID: "my-schema-123",
			schemaClient: func(t *testing.T) *SchemaClient {
				t.Helper()

				c := newMockPubSubClient(t)
				c.EXPECT().GetSchema(ctx, &eventbusv1.SchemaRequest{
					SchemaId: "my-schema-123",
				}).Return(&eventbusv1.SchemaInfo{
					SchemaId:   "my-schema-123",
					SchemaJson: "not-schema",
				}, nil)

				return newSchemaClient(c)
			},
			wantErr: errors.New(`failed to parse schema "my-schema-123": avro: unknown type: not-schema`),
		},
		{
			name:     "fail to unmarshal",
			schemaID: "my-schema-123",
			schemaClient: func(t *testing.T) *SchemaClient {
				t.Helper()

				c := newMockPubSubClient(t)
				c.EXPECT().GetSchema(ctx, &eventbusv1.SchemaRequest{
					SchemaId: "my-schema-123",
				}).Return(&eventbusv1.SchemaInfo{
					SchemaId:   "my-schema-123",
					SchemaJson: testSchema,
				}, nil)

				return newSchemaClient(c)
			},
			data:    []byte("foobar"),
			wantErr: errors.New(`failed to unmarshal with schema "my-schema-123": map[string]interface {}: avro: ReadString: invalid string length`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			data, err := tc.schemaClient(t).Unmarshal(ctx, tc.schemaID, tc.data)
			if tc.wantErr != nil {
				is.True(err != nil)
				is.Equal(err.Error(), tc.wantErr.Error())
			} else {
				is.NoErr(err)
				is.Equal("", cmp.Diff(data, tc.expected))
			}
		})
	}
}

func Test_SchemaMarshal(t *testing.T) {
	ctx := context.Background()
	decoded, encoded := encodedTestData(t, testSchema)

	tests := []struct {
		name         string
		schemaClient func(t *testing.T) *SchemaClient
		schemaID     string
		data         map[string]any
		expected     []byte
		wantErr      error
	}{
		{
			name:     "marshal success",
			schemaID: "my-schema-123",
			schemaClient: func(t *testing.T) *SchemaClient {
				t.Helper()

				c := newMockPubSubClient(t)
				c.EXPECT().GetSchema(ctx, &eventbusv1.SchemaRequest{
					SchemaId: "my-schema-123",
				}).Return(&eventbusv1.SchemaInfo{
					SchemaId:   "my-schema-123",
					SchemaJson: testSchema,
				}, nil)

				return newSchemaClient(c)
			},
			data:     decoded,
			expected: encoded,
		},
		{
			name:     "fail to get schema",
			schemaID: "my-schema-123",
			schemaClient: func(t *testing.T) *SchemaClient {
				t.Helper()

				c := newMockPubSubClient(t)
				c.EXPECT().GetSchema(ctx, &eventbusv1.SchemaRequest{
					SchemaId: "my-schema-123",
				}).Return(nil, errors.New("boom"))
				return newSchemaClient(c)
			},
			wantErr: errors.New(`failed to retrieve schema "my-schema-123": boom`),
		},

		{
			name:     "fail to marshal",
			schemaID: "my-schema-123",
			schemaClient: func(t *testing.T) *SchemaClient {
				t.Helper()

				c := newMockPubSubClient(t)
				c.EXPECT().GetSchema(ctx, &eventbusv1.SchemaRequest{
					SchemaId: "my-schema-123",
				}).Return(&eventbusv1.SchemaInfo{
					SchemaId:   "my-schema-123",
					SchemaJson: testSchema,
				}, nil)

				return newSchemaClient(c)
			},
			data:    map[string]any{"foo": "bar"},
			wantErr: errors.New(`failed to marshal with schema "my-schema-123": avro: missing required field CreatedDate`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			v, err := tc.schemaClient(t).Marshal(ctx, tc.schemaID, tc.data)
			if tc.wantErr != nil {
				is.True(err != nil)
				is.Equal(err.Error(), tc.wantErr.Error())
			} else {
				is.NoErr(err)
				is.Equal("", cmp.Diff(v, tc.expected))
			}
		})
	}
}

func encodedTestData(t *testing.T, schema string) (map[string]any, []byte) {
	t.Helper()

	is := is.New(t)
	sch, err := avro.Parse(schema)
	is.NoErr(err)

	r := map[string]any{
		"Case_Status__c":  "started",
		"CreatedById":     "0012024",
		"CreatedDate":     int64(1746202771),
		"RandomNumber__c": float64(2021),
	}

	data, err := avro.Marshal(sch, r)
	is.NoErr(err)

	return r, data
}
