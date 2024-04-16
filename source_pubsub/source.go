package source

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conduitio-labs/conduit-connector-salesforce/pubsub/proto"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

//go:generate paramgen -output=paramgen_src.go Config

type Config struct {
	// ClientID is the client id from the salesforce app
	ClientID string `json:"clientID" validate:"required"`

	// ClientSecret is the client secret from the salesforce app
	ClientSecret string `json:"clientSecret" validate:"required"`

	// Username is the client secret from the salesforce app
	Username string `json:"username" validate:"required"`

	// OAuthEndpoint is the OAuthEndpoint from the salesforce app
	OAuthEndpoint string `json:"oauthEndpoint" validate:"required"`

	// TopicName is the topic the source connector will subscribe to
	TopicName string `json:"topicName" validate:"required"`
}

type Source struct {
	sdk.UnimplementedSource

	client          *PubSubClient
	subscribeClient proto.PubSub_SubscribeClient
	currReplayId    []byte
	config          Config
}

func New() sdk.Source {
	return sdk.SourceWithMiddleware(&Source{}, sdk.DefaultSourceMiddleware()...)
}

func (s *Source) Parameters() map[string]sdk.Parameter {
	return s.config.Parameters()
}

func (s *Source) Configure(ctx context.Context, cfg map[string]string) error {
	if err := sdk.Util.ParseConfig(cfg, &s.config); err != nil {
		return fmt.Errorf("Failed to parse config")
	}

	sdk.Logger(ctx).Info().Msg("parsed source configuration")

	return nil
}

func (s *Source) Open(ctx context.Context, sdkPos sdk.Position) (err error) {
	s.client, err = NewGRPCClient()
	if err != nil {
		return fmt.Errorf("could not create GRPCClient: %w", err)
	}

	sdk.Logger(ctx).Info().Msg("created GRPCClient")

	creds := Credentials{
		ClientID:      s.config.ClientID,
		ClientSecret:  s.config.ClientSecret,
		OAuthEndpoint: s.config.OAuthEndpoint,
	}

	if err := s.client.Authenticate(creds); err != nil {
		return fmt.Errorf("could not authenticate: %w", err)
	}
	sdk.Logger(ctx).Info().Msg("authenticated GRPCClient")

	err = s.client.FetchUserInfo(s.config.OAuthEndpoint)
	if err != nil {
		return fmt.Errorf("could not fetch user info: %w", err)
	}
	sdk.Logger(ctx).Info().Msg("fetched user info")

	topic, err := s.client.GetTopic(s.config.TopicName)
	if err != nil {
		return fmt.Errorf("could not fetch topic: %w", err)
	}
	sdk.Logger(ctx).Info().Msgf("got topic %s", topic.TopicName)

	if !topic.GetCanSubscribe() {
		return fmt.Errorf("this user is not allowed to subscribe to the following topic: %s", topic.TopicName)
	}

	s.subscribeClient, s.currReplayId, err = s.client.Subscribe(
		ctx,
		s.config.TopicName,
		proto.ReplayPreset_LATEST,
		nil)
	if err != nil {
		return fmt.Errorf("could not subscribe to topic")
	}
	sdk.Logger(ctx).Info().Msgf("subscribed to topic %s", topic.TopicName)

	return nil
}

func (s *Source) Read(ctx context.Context) (rec sdk.Record, err error) {
	recvEvents, currReplayId, err := s.client.Recv(ctx, s.subscribeClient, s.currReplayId)
	if err != nil {
		return rec, err
	}
	s.currReplayId = currReplayId

	bs, err := json.Marshal(map[string]any{"events": recvEvents})
	if err != nil {
		return rec, fmt.Errorf("failed to marshal events")
	}

	var (
		position sdk.Position
		metadata sdk.Metadata
		key      sdk.Data = sdk.RawData(currReplayId)
		payload  sdk.Data = sdk.RawData(bs)
	)

	rec = sdk.Util.Source.NewRecordCreate(position, metadata, key, payload)
	return rec, nil
}

func (s *Source) Ack(ctx context.Context, position sdk.Position) error {
	return nil
}

func (s *Source) Teardown(ctx context.Context) error {
	if err := s.subscribeClient.CloseSend(); err != nil {
		return err
	}

	s.client.Close()
	return nil
}
