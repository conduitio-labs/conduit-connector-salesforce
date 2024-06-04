// Copyright © 2022 Meroxa, Inc.
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

	sdk "github.com/conduitio/conduit-connector-sdk"
)

//go:generate mockery --with-expecter --name=client --inpackage --log-level error
type client interface {
	Next(context.Context) (sdk.Record, error)
	Initialize(context.Context, Config) error
	ReplayID() []byte
	Stop()
	Close() error
	Wait(context.Context) error
}

var _ client = (*PubSubClient)(nil)

type Source struct {
	sdk.UnimplementedSource

	client client
	config Config
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

	if err := s.config.Validate(); err != nil {
		return fmt.Errorf("config failed to validate: %w", err)
	}

	sdk.Logger(ctx).Info().Msg("parsed source configuration")

	return nil
}

func (s *Source) Open(ctx context.Context, sdkPos sdk.Position) error {
	sdk.Logger(ctx).Debug().Msg("Open - Open Connector")

	client, err := NewGRPCClient(ctx, s.config, sdkPos)
	if err != nil {
		return fmt.Errorf("could not create GRPCClient: %w", err)
	}

	if err := client.Initialize(ctx, s.config); err != nil {
		return fmt.Errorf("could not initialize pubsub client: %w", err)
	}

	s.client = client

	return nil
}

func (s *Source) Read(ctx context.Context) (rec sdk.Record, err error) {
	sdk.Logger(ctx).Debug().Msg("Read - Getting next event")

	r, err := s.client.Next(ctx)
	if err != nil {
		sdk.Logger(ctx).Error().Err(err).Msg("next: failed to get next record")
		return sdk.Record{}, err
	}

	sdk.Logger(ctx).Debug().Msgf("read event: %+v", r)

	// filter out empty record payloads
	if r.Payload.Before == nil && r.Payload.After == nil {
		sdk.Logger(ctx).Error().
			Msgf("empty record payload detected. backing off: %+v", r)

		return sdk.Record{}, sdk.ErrBackoffRetry
	}

	return r, nil
}

func (s *Source) Ack(ctx context.Context, position sdk.Position) error {
	sdk.Logger(ctx).Debug().Str("position", string(position)).Msg("got ack")
	return nil
}

func (s *Source) Teardown(ctx context.Context) error {
	s.client.Stop()

	if err := s.client.Wait(ctx); err != nil {
		sdk.Logger(ctx).Error().Err(err).
			Msg("received error while stopping client")
	}

	if err := s.client.Close(); err != nil {
		return fmt.Errorf("error when closing subscriber conn: %w", err)
	}

	return nil
}
