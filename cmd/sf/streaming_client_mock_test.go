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

package main

import (
	"context"
	"sync"

	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/miquido/conduit-connector-salesforce/internal/cometd/responses"
)

type streamingClientMock struct {
	results   []sdk.Record
	lastIndex int
	mutex     sync.Mutex
}

func (s *streamingClientMock) SetResults(results []sdk.Record) {
	s.mutex.Lock()
	s.results = results
	s.mutex.Unlock()
}

func (s *streamingClientMock) Handshake(_ context.Context) (responses.SuccessfulHandshakeResponse, error) {
	// Make Handshake always successful
	return responses.SuccessfulHandshakeResponse{
		Successful: true,
	}, nil
}

func (s *streamingClientMock) Connect(_ context.Context) (responses.ConnectResponse, error) {
	s.mutex.Lock()

	response := responses.ConnectResponse{
		Successful: true,
		Events:     make([]responses.ConnectResponseEvent, 0, len(s.results)),
	}

	for _, record := range s.results {
		response.Events = append(response.Events, responses.ConnectResponseEvent{
			Data: responses.ConnectResponseEventData{
				Event: responses.ConnectResponseEventDataMetadata{
					CreatedDate: record.CreatedAt,
					ReplayID:    s.lastIndex,
				},
				SObject: record.Payload.(sdk.StructuredData),
			},
			Channel: "MyTopic1",
		})

		s.lastIndex++
	}

	s.results = nil

	s.mutex.Unlock()

	// Make Connect always successful
	return response, nil
}

func (s *streamingClientMock) SubscribeToPushTopic(_ context.Context, pushTopic string) (responses.SubscribeResponse, error) {
	// Make SubscribeToPushTopic always successful
	return responses.SubscribeResponse{
		Successful:   true,
		Subscription: []string{pushTopic},
	}, nil
}

func (s *streamingClientMock) UnsubscribeToPushTopic(ctx context.Context, pushTopic string) (responses.UnsubscribeResponse, error) {
	// Make UnsubscribeToPushTopic always successful
	return responses.UnsubscribeResponse{
		Successful: true,
	}, nil
}

func (s *streamingClientMock) Disconnect(ctx context.Context) (responses.DisconnectResponse, error) {
	// Make Disconnect always successful
	return responses.DisconnectResponse{
		Successful: true,
	}, nil
}
