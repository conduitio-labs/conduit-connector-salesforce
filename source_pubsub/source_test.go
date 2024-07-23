// Copyright © 2022 Meroxa, Inc. and Miquido
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
	"testing"

	sdk "github.com/conduitio/conduit-connector-sdk"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Read(t *testing.T) {
	testRecord := sdk.Record{
		Position:  []byte("test1"),
		Operation: sdk.OperationCreate,
		Metadata: sdk.Metadata{
			"test1":              "test",
			"opencdc.collection": "test",
		},
		Key: sdk.StructuredData{
			"test1": "test",
		},
		Payload: sdk.Change{
			After: sdk.StructuredData{
				"test1": "test",
			},
		},
	}

	testConfig := Config{
		ClientID:      "test-client-id",
		ClientSecret:  "test-client-secret",
		OAuthEndpoint: "https://somewhere",
		TopicNames:    []string{"/events/TestEvent__e", "/events/TestEvent2__e"},
	}

	testCases := []struct {
		desc           string
		config         Config
		mockClient     func() *mockClient
		expectedRecord sdk.Record
		expectedErr    error
	}{
		{
			desc:   "success - receive event",
			config: testConfig,
			mockClient: func() *mockClient {
				m := newMockClient(t)
				m.On("Next", mock.Anything).Return(testRecord, nil)

				return m
			},
			expectedRecord: testRecord,
		},
		{
			desc:   "success - no event, backoff",
			config: testConfig,
			mockClient: func() *mockClient {
				m := newMockClient(t)
				m.On("Next", mock.Anything).Return(sdk.Record{}, nil).Times(1)
				return m
			},
			expectedErr: sdk.ErrBackoffRetry,
		},

		{
			desc:   "error - failed on Next",
			config: testConfig,
			mockClient: func() *mockClient {
				m := newMockClient(t)
				m.On("Next", mock.Anything).Return(sdk.Record{}, errors.New("error receiving new events - test error")).Times(1)
				return m
			},
			expectedErr: errors.New("error receiving new events - test error"),
		},
		{
			desc:   "error - record with empty payload",
			config: testConfig,
			mockClient: func() *mockClient {
				m := newMockClient(t)
				m.On("Next", mock.Anything).Return(sdk.Record{Payload: sdk.Change{Before: nil, After: nil}}, nil).Times(1)
				return m
			},
			expectedErr: sdk.ErrBackoffRetry,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			s := Source{
				config: tc.config,
			}
			if tc.mockClient != nil {
				s.client = tc.mockClient()
			}

			r, err := s.Read(ctx)
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedRecord, r)
			}
		})
	}
}