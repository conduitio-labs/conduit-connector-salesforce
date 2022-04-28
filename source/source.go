package source

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/conduitio/conduit-connector-salesforce/internal/cometd"
	"github.com/conduitio/conduit-connector-salesforce/internal/cometd/responses"
	"github.com/conduitio/conduit-connector-salesforce/internal/salesforce/oauth"
	sdk "github.com/conduitio/conduit-connector-sdk"
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
	fmt.Printf("Configure")

	s.config, err = ParseConfig(cfgRaw)

	return
}

func (s *Source) Open(ctx context.Context, _ sdk.Position) error {
	fmt.Printf("Open")

	// Authenticate
	oAuthClient := oauth.NewClient(
		s.config.Environment,
		s.config.ClientId,
		s.config.ClientSecret,
		s.config.Username,
		s.config.Password,
		s.config.SecurityToken,
	)

	token, err := oAuthClient.Authenticate()
	if err != nil {
		return fmt.Errorf("could not authenticate: %w", err)
	}

	// Streaming API client
	s.streamingClient, err = cometd.NewClient(
		fmt.Sprintf("%s/cometd/%s", token.InstanceUrl, sfCometDVersion),
		token.AccessToken,
	)
	if err != nil {
		return fmt.Errorf("could not create Streaming API client: %w", err)
	}

	// Handshake
	if _, err := s.streamingClient.Handshake(); err != nil {
		return fmt.Errorf("handshake error: %w", err)
	}

	// Subscribe to topic
	subscribeResponse, err := s.streamingClient.SubscribeToPushTopic(s.config.PushTopicName)
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
	fmt.Printf("Read")

	select {
	case event, ok := <-s.events:
		if !ok {
			return sdk.Record{}, fmt.Errorf("connection closed by the server")
		}

		return sdk.Record{
			CreatedAt: event.Data.Event.CreatedDate,
			Payload:   sdk.StructuredData(event.Data.Sobject),
			Metadata: map[string]string{
				"channel":   event.Channel,
				"replayId":  strconv.FormatInt(int64(event.Data.Event.ReplayId), 10),
				"eventType": event.Data.Event.Type,
			},
		}, nil

	case err := <-s.errors:
		return sdk.Record{}, err

	case <-ctx.Done():
		return sdk.Record{}, ctx.Err()
	}
}

func (s *Source) Ack(_ context.Context, _ sdk.Position) error {
	fmt.Printf("Ack")

	return nil // no ack needed
}

func (s *Source) Teardown(ctx context.Context) error {
	fmt.Printf("Teardown")

	// Unsubscribe
	unsubscribeResponse, err := s.streamingClient.UnsubscribeToPushTopic(s.config.PushTopicName)
	if err != nil {
		sdk.Logger(ctx).Warn().Msgf("unsubscribe error: %s", err)
	} else if !unsubscribeResponse.Successful {
		sdk.Logger(ctx).Warn().Msgf("unsubscribe error: %s", unsubscribeResponse.Error)
	}

	// Disconnect
	disconnectResponse, err := s.streamingClient.Disconnect()
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
		connectResponse, err := s.streamingClient.Connect()
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
				if _, err := s.streamingClient.Handshake(); err != nil {
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
