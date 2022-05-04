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

	return
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
		return fmt.Errorf("could not authenticate: %w", err)
	}

	// Streaming API client
	s.streamingClient, err = cometd.NewClient(
		fmt.Sprintf("%s/cometd/%s", token.InstanceURL, sfCometDVersion),
		token.AccessToken,
	)
	if err != nil {
		return fmt.Errorf("could not create Streaming API client: %w", err)
	}

	// Handshake
	if _, err := s.streamingClient.Handshake(ctx); err != nil {
		return fmt.Errorf("handshake error: %w", err)
	}

	// Subscribe to topic
	subscribeResponse, err := s.streamingClient.SubscribeToPushTopic(ctx, s.config.PushTopicName)
	if err != nil {
		return fmt.Errorf("subscribe error: %w", err)
	}
	if !subscribeResponse.Successful {
		return fmt.Errorf("subscribe error: %s", subscribeResponse.Error)
	}

	// Register subscriptions that we should listen to
	for _, sub := range subscribeResponse.GetSubscriptions() {
		s.subscriptions[sub] = true
	}

	// Start events worker
	go func() {
		s.eventsWorker(ctx)
	}()

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

		return sdk.Record{
			Key:       keyValue,
			Payload:   sdk.StructuredData(event.Data.Sobject),
			Position:  sdk.Position(keyValue),
			CreatedAt: event.Data.Event.CreatedDate,
			Metadata: map[string]string{
				"channel":  event.Channel,
				"replayId": strconv.FormatInt(int64(event.Data.Event.ReplayID), 10),
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
	unsubscribeResponse, err := s.streamingClient.UnsubscribeToPushTopic(ctx, s.config.PushTopicName)
	if err != nil {
		sdk.Logger(ctx).Warn().Msgf("unsubscribe error: %s", err)
	} else if !unsubscribeResponse.Successful {
		sdk.Logger(ctx).Warn().Msgf("unsubscribe error: %s", unsubscribeResponse.Error)
	}

	// Disconnect
	disconnectResponse, err := s.streamingClient.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("disconnect error: %w", err)
	}
	if !disconnectResponse.Successful {
		return fmt.Errorf("disconnect error: %s", disconnectResponse.Error)
	}

	s.subscriptions = nil
	s.streamingClient = nil

	return nil
}

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

func (s *Source) getKeyValue(event responses.ConnectResponseEvent) (sdk.RawData, error) {
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
