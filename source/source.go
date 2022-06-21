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
	"fmt"
	"strconv"
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/miquido/conduit-connector-salesforce/internal"
	"github.com/miquido/conduit-connector-salesforce/internal/cometd"
	"github.com/miquido/conduit-connector-salesforce/internal/cometd/responses"
	"github.com/miquido/conduit-connector-salesforce/internal/salesforce/oauth"
	"gopkg.in/tomb.v2"
)

const sfCometDVersion = "54.0"

var OAuthClientFactory = oauth.NewDefaultClient
var StreamingClientFactory = cometd.NewDefaultClient

var ErrConnectorIsStopped = errors.New("connector is stopped")

type Source struct {
	sdk.UnimplementedSource

	config          Config
	streamingClient cometd.Client
	subscriptions   map[string]bool
	events          chan responses.ConnectResponseEvent
	tomb            *tomb.Tomb
}

func NewSource() sdk.Source {
	return &Source{
		subscriptions: make(map[string]bool),
		events:        make(chan responses.ConnectResponseEvent),
		tomb:          nil,
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
	oAuthClient := OAuthClientFactory(
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
	s.streamingClient, err = StreamingClientFactory(
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
	s.tomb = &tomb.Tomb{}
	s.tomb.Go(s.eventsWorker)

	return nil
}

func (s *Source) Read(ctx context.Context) (sdk.Record, error) {
	if s.tomb == nil {
		return sdk.Record{}, ErrConnectorIsStopped
	}

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

		var action internal.Operation

		switch event.Data.Event.Type {
		case responses.CreatedEventType:
			action = internal.OperationInsert

		case responses.UpdatedEventType, responses.UndeletedEventType:
			action = internal.OperationUpdate

		case responses.DeletedEventType:
			action = internal.OperationDelete

		default:
			sdk.Logger(ctx).Info().Msgf(
				"unknown event type: %q, falling back to %q",
				event.Data.Event.Type,
				internal.OperationInsert,
			)

			action = internal.OperationInsert
		}

		return sdk.Record{
			Key:       keyValue,
			Payload:   sdk.StructuredData(event.Data.SObject),
			Position:  sdk.Position(replayID),
			CreatedAt: event.Data.Event.CreatedDate,
			Metadata: map[string]string{
				"channel":  event.Channel,
				"replayId": replayID,
				"action":   action,
			},
		}, nil

	case <-s.tomb.Dead():
		err := s.tomb.Err()
		if err == nil {
			err = ErrConnectorIsStopped
		}

		return sdk.Record{}, err

	case <-ctx.Done():
		return sdk.Record{}, ctx.Err()
	}
}

func (s *Source) Ack(ctx context.Context, position sdk.Position) error {
	sdk.Logger(ctx).Debug().Str("position", string(position)).Msg("got ack")

	return nil // no ack needed
}

func (s *Source) Teardown(ctx context.Context) error {
	var err error

	if s.tomb != nil {
		s.tomb.Kill(ErrConnectorIsStopped)

		err = s.tomb.Wait()

		// Worker was properly closed
		if errors.Is(err, ErrConnectorIsStopped) {
			err = nil
		}
	}

	if s.streamingClient != nil {
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

		// Close the streaming client
		s.streamingClient = nil
	}

	// Remove registered subscriptions and free the memory
	s.subscriptions = nil

	return err
}

// eventsWorker continuously queries for data updates from Salesforce
func (s *Source) eventsWorker() error {
	defer close(s.events)

	for {
		select {
		case <-s.tomb.Dying():
			return s.tomb.Err()

		default:
			ctx := s.tomb.Context(context.Background())

			// Receive event
			connectResponse, err := s.streamingClient.Connect(ctx)
			if err != nil {
				return fmt.Errorf("failed to receive event: %w", err)
			}

			// If not successful, check how to retry
			if !connectResponse.Successful {
				if nil == connectResponse.Advice {
					return fmt.Errorf("failed to receive event and no reconnection strategy provided by the server: %s", connectResponse.Error)
				}

				switch connectResponse.Advice.Reconnect {
				case responses.AdviceReconnectRetry:
					// Check if request can be retried
					if connectResponse.Advice.Interval < 0 {
						return fmt.Errorf("server disallowed for reconnect, stopping")
					}

					// Wait and retry
					time.Sleep(time.Millisecond * time.Duration(connectResponse.Advice.Interval))

					continue

				case responses.AdviceReconnectHandshake:
					// Handshake and retry
					if _, err := s.streamingClient.Handshake(ctx); err != nil {
						return fmt.Errorf("reconnect handshake error: %w", err)
					}

					continue

				case responses.AdviceReconnectNone:
					// Cannot retry
					return fmt.Errorf("server disallowed for reconnect, stopping")

				default:
					// Unexpected, cannot retry
					return fmt.Errorf("unsupported reconnect advice: %s", connectResponse.Advice.Reconnect)
				}
			}

			// If successful, send event
			for _, event := range connectResponse.Events {
				if _, exists := s.subscriptions[event.Channel]; exists {
					// Send out the record if possible
					select {
					case s.events <- event:
						// sdk.Record was sent successfully

					case <-s.tomb.Dying():
						return s.tomb.Err()
					}
				} else {
					sdk.Logger(ctx).Debug().Msgf("Received event for unsupported channel: %s", event.Channel)
				}
			}
		}
	}
}

// getKeyValue prepares the Key value for Payload
func (s *Source) getKeyValue(event responses.ConnectResponseEvent) (sdk.RawData, error) {
	if s.config.KeyField == "" {
		return nil, nil
	}

	value, exists := event.Data.SObject[s.config.KeyField]
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
