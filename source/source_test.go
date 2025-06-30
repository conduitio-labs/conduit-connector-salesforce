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

package source

import (
	"context"
	"errors"
	"strings"
	"testing"

	config "github.com/conduitio-labs/conduit-connector-salesforce/config"
	"github.com/conduitio-labs/conduit-connector-salesforce/internal/eventbus"
	"github.com/conduitio-labs/conduit-connector-salesforce/source/position"
	opencdc "github.com/conduitio/conduit-commons/opencdc"
	"github.com/matryer/is"
	"github.com/stretchr/testify/require"
)

func Test_Read(t *testing.T) {
	is := is.New(t)

	expectedRecord := opencdc.Record{
		Position:  []byte(`topics`),
		Operation: opencdc.OperationCreate,
		Metadata: opencdc.Metadata{
			"opencdc.collection": "test",
		},
		Key: opencdc.StructuredData{
			"id":       "event-1",
			"replayId": []byte(`replay-id`),
		},
		Payload: opencdc.Change{
			After: opencdc.StructuredData{
				"test1": "test",
			},
		},
	}

	config := config.Config{
		ClientID:      "test-client-id",
		ClientSecret:  "test-client-secret",
		OAuthEndpoint: "https://somewhere",
	}

	testConfig := Config{
		Config:     config,
		TopicNames: []string{"/events/TestEvent__e", "/events/TestEvent2__e"},
	}

	testCases := []struct {
		desc           string
		config         Config
		context        func() context.Context
		iterator       func() *iterator
		expectedRecord opencdc.Record
		expectedErr    error
	}{
		{
			desc:   "successful iteration",
			config: testConfig,
			iterator: func() *iterator {
				testEvent := &eventbus.EventData{
					Topic:    "test",
					Data:     map[string]interface{}{"test1": "test"},
					ID:       "event-1",
					ReplayID: []byte(`replay-id`),
				}
				return newTestIterator([]*eventbus.EventData{testEvent})
			},
			expectedRecord: expectedRecord,
		},
		{
			desc:   "error on iterator stopped",
			config: testConfig,
			iterator: func() *iterator {
				iterator := &iterator{
					events:       make(chan *eventbus.EventData),
					lastPosition: position.New([]string{"topic1"}),
				}

				return iterator
			},
			expectedErr: errors.New("iterator stopped: context canceled"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := Source{
				config: tc.config,
			}
			if tc.iterator != nil {
				s.i = tc.iterator()
			}
			if tc.expectedErr != nil {
				cancel()
			}

			r, err := s.Read(ctx)
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				cancel()
				require.NoError(t, err)
				is.True(strings.Contains(
					string(r.Position),
					`{"topics":{"test":{"replayID":"cmVwbGF5LWlk",`),
				) // skip matching the read time
				is.True(r.Payload.Before == nil)
				is.True(r.Payload.After != nil)
				is.Equal(r.Metadata["opencdc.collection"], tc.expectedRecord.Metadata["opencdc.collection"])
				is.Equal(r.Payload.After, tc.expectedRecord.Payload.After)
			}
		})
	}
}

func newTestIterator(events []*eventbus.EventData) *iterator {
	i := &iterator{
		events:       make(chan *eventbus.EventData, len(events)),
		lastPosition: position.New([]string{"topic1"}),
	}

	for _, e := range events {
		i.events <- e
	}
	close(i.events)

	return i
}
