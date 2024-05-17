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
		Position: []byte("test1"),
		Operation: sdk.OperationCreate,
		Metadata: sdk.Metadata{
			"test1": "test",
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
		ClientID: "test-client-id",
		ClientSecret: "test-client-secret",
		OAuthEndpoint: "https://somewhere",
		TopicName: "/events/TestEvent__e",
	}

	disconnectErr := errors.New("upstream connect error or disconnect/reset before headers. reset reason: connection termination")

	testCases := []struct{
		desc string
		config Config
		mockClient func() *mockClient
		expectedRecord sdk.Record
		expectedErr error
	}{
		{
			desc: "success - receive event",
			config: testConfig,
			mockClient: func() *mockClient{
				m := newMockClient(t)
				m.On("HasNext", mock.Anything).Return(true)
				m.On("Next", mock.Anything).Return(testRecord, nil)
				return m
			},
			expectedRecord: testRecord,
		},
		{
			desc: "success - no event, backoff",
			config: testConfig,
			mockClient: func() *mockClient{
				m := newMockClient(t)
				m.On("HasNext", mock.Anything).Return(false)
				return m
			},
			expectedErr: sdk.ErrBackoffRetry,
		},
		{
			desc: "success after reconnecting",
			config: testConfig,
			mockClient: func() *mockClient{
				m := newMockClient(t)
				m.On("HasNext", mock.Anything).Return(true).Times(1)
				m.On("Next", mock.Anything).Return(sdk.Record{}, disconnectErr).Times(1)
				m.On("Initialize", mock.Anything, testConfig).Return(nil).Times(1)
				m.On("Next", mock.Anything).Return(testRecord, nil).Times(1)
				return m
			},
			expectedRecord: testRecord,
		},
		{
			desc: "failed on Next after reconnect",
			config: testConfig,
			mockClient: func() *mockClient{
				m := newMockClient(t)
				m.On("HasNext", mock.Anything).Return(true).Times(1)
				m.On("Next", mock.Anything).Return(sdk.Record{}, disconnectErr).Times(1)
				m.On("Initialize", mock.Anything, testConfig).Return(nil).Times(1)
				m.On("Next", mock.Anything).Return(sdk.Record{}, errors.New("test error")).Times(1)
				return m
			},
			expectedErr: errors.New("error receiving new events - test error"),
		},
		{
			desc: "error - failed to reconnect",
			config: testConfig,
			mockClient: func() *mockClient{
				m := newMockClient(t)
				m.On("HasNext", mock.Anything).Return(true).Times(1)
				m.On("Next", mock.Anything).Return(sdk.Record{}, disconnectErr).Times(1)
				m.On("Initialize", mock.Anything, testConfig).Return(errors.New("test error")).Times(1)
				return m
			},
			expectedErr: errors.New("error reinitializing client - test error"),
		},
		{
			desc: "error - failed on Next",
			config: testConfig,
			mockClient: func() *mockClient{
				m := newMockClient(t)
				m.On("HasNext", mock.Anything).Return(true).Times(1)
				m.On("Next", mock.Anything).Return(sdk.Record{}, errors.New("test error")).Times(1)
				return m
			},
			expectedErr: errors.New("error receiving new events - test error"),
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