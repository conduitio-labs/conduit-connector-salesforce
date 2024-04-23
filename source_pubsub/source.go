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
	"time"

	"github.com/conduitio-labs/conduit-connector-salesforce/source_pubsub/proto"
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
	config          Config
}

func NewSource() sdk.Source {
	return sdk.SourceWithMiddleware(&Source{}, sdk.DefaultSourceMiddleware()...)
}

func (s *Source) Parameters() map[string]sdk.Parameter {
	return s.config.Parameters()
}

func (s *Source) Configure(ctx context.Context, cfg map[string]string) error {
	if err := sdk.Util.ParseConfig(cfg, &s.config); err != nil {
		return fmt.Errorf("failed to parse config - %s", err)
	}

	sdk.Logger(ctx).Info().Msg("parsed source configuration")

	return nil
}

func (s *Source) Open(ctx context.Context, sdkPos sdk.Position) (err error) {
	sdk.Logger(ctx).Debug().Msg("Open - Open Connector")

	s.client, err = NewGRPCClient(time.Duration(2), s.config, sdkPos)
	if err != nil {
		return fmt.Errorf("could not create GRPCClient: %w", err)
	}

	return nil
}

func (s *Source) Read(ctx context.Context) (rec sdk.Record, err error) {
	sdk.Logger(ctx).Debug().Msg("Read - Start reading events")
	if !s.client.HasNext(ctx) {
		sdk.Logger(ctx).Debug().Msg("Read - No next events, backoff....")
		return sdk.Record{}, sdk.ErrBackoffRetry
	}
	sdk.Logger(ctx).Debug().Msg("Read - Getting next event.")
	r, err := s.client.Next(ctx)
	if err != nil {
		return sdk.Record{}, fmt.Errorf("error receiving new events - %s", err)
	}
	return r, nil
}

func (s *Source) Ack(ctx context.Context, position sdk.Position) error {
	sdk.Logger(ctx).Debug().Str("position", string(position)).Msg("got ack")
	return nil
}

func (s *Source) Teardown(_ context.Context) error {
	if err := s.subscribeClient.CloseSend(); err != nil {
		return err
	}

	s.client.Close()
	return nil
}
