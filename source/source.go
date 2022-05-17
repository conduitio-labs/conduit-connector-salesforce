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
	"fmt"
	"strconv"
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/miquido/conduit-connector-salesforce/internal/cometd"
	"github.com/miquido/conduit-connector-salesforce/internal/cometd/responses"
	"github.com/miquido/conduit-connector-salesforce/internal/salesforce/oauth"
)

const sfCometDVersion = "54.0"

type Source struct {
	sdk.UnimplementedSource

	config          Config
	streamingClient *cometd.Client
	subscriptions   map[string]bool
	events          chan responses.ConnectResponseEvent
	errors          chan error
}

func NewSource() sdk.Source {
	return &Source{
		subscriptions: make(map[string]bool),
		events:        make(chan responses.ConnectResponseEvent),
		errors:        make(chan error),
	}
}

func (s *Source) Configure(_ context.Context, cfgRaw map[string]string) (err error) {
	s.config, err = ParseConfig(cfgRaw)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	return nil
}

func (s *Source) Open(ctx context.Context, _ sdk.Position) error {
	// Authenticate
	oAuthClient := oauth.NewClient(
		s.config.Environment,
		s.config.ClientID,
		s.config.ClientSecret,
		s.config.Username,
		s.config.Password,
		s.config.SecurityToken,
	)

	token, err := oAuthClient.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("connector open error: could not authenticate: %w", err)
	}

	// Streaming API client
	s.streamingClient, err = cometd.NewClient(
		fmt.Sprintf("%s/cometd/%s", token.InstanceURL, sfCometDVersion),
		token.AccessToken,
	)
	if err != nil {
		return fmt.Errorf("connector open error: could not create Streaming API client: %w", err)
	}

	// Handshake
	if _, err := s.streamingClient.Handshake(ctx); err != nil {
		return fmt.Errorf("connector open error: handshake error: %w", err)
	}

	// Subscribe to topic
	for _, pushTopicName := range s.config.PushTopicsNames {
		subscribeResponse, err := s.streamingClient.SubscribeToPushTopic(ctx, pushTopicName)
		if err != nil {
			return fmt.Errorf("connector open error: subscribe error: failed to subscribe %q topic: %w", pushTopicName, err)
		}
		if !subscribeResponse.Successful {
			return fmt.Errorf("connector open error: subscribe error: failed to subscribe %q topic: %s", pushTopicName, subscribeResponse.Error)
		}

		// Register subscriptions that we should listen to
		for _, sub := range subscribeResponse.GetSubscriptions() {
			s.subscriptions[sub] = true
		}
	}

	// Start events worker
	go func(ctx context.Context) {
		s.eventsWorker(ctx)
	}(ctx)

	return nil
}

func (s *Source) Read(ctx context.Context) (sdk.Record, error) {
	select {
	case event, ok := <-s.events:
		if !ok {
			return sdk.Record{}, fmt.Errorf("connection closed by the server")
		}

		keyValue, err := s.getKeyValue(event)
		if err != nil {
			return sdk.Record{}, err
		}

		replayID := strconv.FormatInt(int64(event.Data.Event.ReplayID), 10)

		return sdk.Record{
			Key:       keyValue,
			Payload:   sdk.StructuredData(event.Data.Sobject),
			Position:  sdk.Position(replayID),
			CreatedAt: event.Data.Event.CreatedDate,
			Metadata: map[string]string{
				"channel":  event.Channel,
				"replayId": replayID,
				"action":   event.Data.Event.Type,
			},
		}, nil

	case err := <-s.errors:
		return sdk.Record{}, err

	case <-ctx.Done():
		return sdk.Record{}, ctx.Err()
	}
}

func (s *Source) Ack(_ context.Context, _ sdk.Position) error {
	return nil // no ack needed
}

func (s *Source) Teardown(ctx context.Context) error {
	// Unsubscribe
	for _, pushTopicName := range s.config.PushTopicsNames {
		unsubscribeResponse, err := s.streamingClient.UnsubscribeToPushTopic(ctx, pushTopicName)
		if err != nil {
			sdk.Logger(ctx).Warn().Msgf("unsubscribe error: failed to unsubscribe %q topic: %s", pushTopicName, err)
		} else if !unsubscribeResponse.Successful {
			sdk.Logger(ctx).Warn().Msgf("unsubscribe error: failed to unsubscribe %q topic: %s", pushTopicName, unsubscribeResponse.Error)
		}
	}

	// Disconnect
	disconnectResponse, err := s.streamingClient.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("connector close error: disconnect error: %w", err)
	}
	if !disconnectResponse.Successful {
		return fmt.Errorf("connector close error: disconnect error: %s", disconnectResponse.Error)
	}

	s.subscriptions = nil
	s.streamingClient = nil

	return nil
}

// eventsWorker continuously queries for data updates from Salesforce
func (s *Source) eventsWorker(ctx context.Context) {
	defer close(s.events)

	for {
		// Receive event
		connectResponse, err := s.streamingClient.Connect(ctx)
		if err != nil {
			s.errors <- fmt.Errorf("failed to receive event: %w", err)

			return
		}

		// If not successful, check how to retry
		if !connectResponse.Successful {
			if nil == connectResponse.Advice {
				s.errors <- fmt.Errorf("failed to receive event and no reconnection strategy provided by the server: %s", connectResponse.Error)

				return
			}

			switch connectResponse.Advice.Reconnect {
			case responses.AdviceReconnectRetry:
				// Check if request can be retried
				if connectResponse.Advice.Interval < 0 {
					s.errors <- fmt.Errorf("server disallowed for reconnect, stopping")

					return
				}

				// Wait and retry
				time.Sleep(time.Millisecond * time.Duration(connectResponse.Advice.Interval))

				continue

			case responses.AdviceReconnectHandshake:
				// Handshake and retry
				if _, err := s.streamingClient.Handshake(ctx); err != nil {
					s.errors <- fmt.Errorf("reconnect handshake error: %w", err)

					return
				}

				continue

			case responses.AdviceReconnectNone:
				// Cannot retry
				s.errors <- fmt.Errorf("server disallowed for reconnect, stopping")

				return

			default:
				// Unexpected, cannot retry
				s.errors <- fmt.Errorf("unsupported reconnect advice: %s", connectResponse.Advice.Reconnect)

				return
			}
		}

		// If successful, send event
		for _, event := range connectResponse.Events {
			if _, exists := s.subscriptions[event.Channel]; exists {
				s.events <- event
			} else {
				sdk.Logger(ctx).Debug().Msgf("Received event for unsupported channel: %s", event.Channel)
			}
		}
	}
}

// getKeyValue prepares the Key value for Payload
func (s *Source) getKeyValue(event responses.ConnectResponseEvent) (sdk.RawData, error) {
	if s.config.KeyField == "" {
		return nil, nil
	}

	value, exists := event.Data.Sobject[s.config.KeyField]
	if !exists {
		return nil, fmt.Errorf("the %q field does not exist in the data", s.config.KeyField)
	}

	switch v := value.(type) {
	case string:
		return sdk.RawData(v), nil

	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return sdk.RawData(fmt.Sprintf("%d", v)), nil

	case float32, float64:
		return sdk.RawData(fmt.Sprintf("%G", v)), nil
	}

	return nil, fmt.Errorf("the %T type of Key field is not supported", value)
}
