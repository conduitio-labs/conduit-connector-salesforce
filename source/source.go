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
	"encoding/base64"
	"fmt"

	"github.com/conduitio-labs/conduit-connector-salesforce/pubsub"
	"github.com/conduitio-labs/conduit-connector-salesforce/source/position"
	"github.com/conduitio/conduit-commons/config"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type client interface {
	Next(context.Context) (opencdc.Record, error)
	Initialize(context.Context) error
	Stop(context.Context)
	Close(context.Context) error
	Wait(context.Context) error
}

var _ client = (*pubsub.Client)(nil)

type Source struct {
	sdk.UnimplementedSource
	client client
	config Config
}

func NewSource() sdk.Source {
	return sdk.SourceWithMiddleware(&Source{}, sdk.DefaultSourceMiddleware()...)
}

func (s *Source) Parameters() config.Parameters {
	return s.config.Parameters()
}

func (s *Source) Configure(ctx context.Context, cfg config.Config) error {
	var c Config

	if err := sdk.Util.ParseConfig(
		ctx,
		cfg,
		&c,
		NewSource().Parameters(),
	); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	c, err := c.Validate(ctx)
	if err != nil {
		return fmt.Errorf("config failed to validate: %w", err)
	}

	s.config = c

	return nil
}

func (s *Source) Open(ctx context.Context, sdkPos opencdc.Position) error {
	logger := sdk.Logger(ctx)

	var parsedPositions position.Topics

	logger.Debug().
		Str("at", "source.open").
		Str("position", base64.StdEncoding.EncodeToString(sdkPos)).
		Strs("topics", s.config.TopicNames).
		Msg("Open Source Connector")

	parsedPositions, err := position.ParseSDKPosition(sdkPos, "")
	if err != nil {
		return fmt.Errorf("error parsing sdk position: %w", err)
	}

	client, err := pubsub.NewGRPCClient(ctx, s.config.Config, parsedPositions)
	if err != nil {
		return fmt.Errorf("could not create GRPCClient: %w", err)
	}

	if err := client.Initialize(ctx); err != nil {
		return fmt.Errorf("could not initialize pubsub client: %w", err)
	}

	s.client = client

	for _, t := range s.config.TopicNames {
		p := parsedPositions.TopicReplayID(t)
		logger.Debug().
			Str("at", "source.open").
			Str("position", string(p)).
			Str("position encoded", base64.StdEncoding.EncodeToString(p)).
			Str("topic", t).
			Msgf("Grpc Client has been set. Will begin read for topic: %s", t)
	}

	return nil
}

func (s *Source) Read(ctx context.Context) (rec opencdc.Record, err error) {
	logger := sdk.Logger(ctx)
	logger.Debug().
		Strs("topics", s.config.TopicNames).
		Msg("begin read")

	r, err := s.client.Next(ctx)
	if err != nil {
		return opencdc.Record{}, fmt.Errorf("failed to get next record: %w", err)
	}

	// filter out empty record payloads
	if r.Payload.Before == nil && r.Payload.After == nil {
		logger.Error().
			Interface("record", r).
			Msg("backing off, empty record payload detected")

		return opencdc.Record{}, sdk.ErrBackoffRetry
	}

	topic, err := r.Metadata.GetCollection()
	if err != nil {
		return opencdc.Record{}, err
	}

	logger.Debug().
		Str("at", "source.read").
		Str("position encoded", base64.StdEncoding.EncodeToString(r.Position)).
		Str("position", string(r.Position)).
		Str("record on topic", topic).
		Msg("sending record")

	return r, nil
}

func (s *Source) Ack(ctx context.Context, pos opencdc.Position) error {
	sdk.Logger(ctx).Debug().
		Str("at", "source.ack").
		Str("uncoded position ", string(pos)).
		Str("position", base64.StdEncoding.EncodeToString(pos)).
		Msg("received ack")

	return nil
}

func (s *Source) Teardown(ctx context.Context) error {
	if s.client == nil {
		return nil
	}

	s.client.Stop(ctx)

	if err := s.client.Wait(ctx); err != nil {
		sdk.Logger(ctx).Error().Err(err).
			Msg("received error while stopping client")
	}

	if err := s.client.Close(ctx); err != nil {
		return fmt.Errorf("error when closing subscriber conn: %w", err)
	}

	return nil
}
