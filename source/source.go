// Copyright © 2024 Meroxa, Inc.
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
	"time"

	pubsub "github.com/conduitio-labs/conduit-connector-salesforce/internal/pubsub"
	"github.com/conduitio-labs/conduit-connector-salesforce/source/position"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
)

type client interface {
	Next(context.Context) (opencdc.Record, error)
	Initialize(context.Context, []string) error
	StartCDC(context.Context, string, position.Topics, []string, time.Duration) error
	Teardown(context.Context) error
}

var _ client = (*pubsub.Client)(nil)

type Source struct {
	sdk.UnimplementedSource
	client client
	config Config
}

func (s *Source) Config() sdk.SourceConfig {
	return &s.config
}

func NewSource() sdk.Source {
	return sdk.SourceWithMiddleware(&Source{})
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
		return errors.Errorf("error parsing sdk position: %w", err)
	}

	client, err := pubsub.NewGRPCClient(s.config.Config, "subscribe")
	if err != nil {
		return errors.Errorf("could not create GRPCClient: %w", err)
	}

	if err := client.Initialize(ctx, s.config.TopicNames); err != nil {
		return errors.Errorf("could not initialize pubsub client: %w", err)
	}

	if err := client.StartCDC(ctx, s.config.ReplayPreset, parsedPositions, s.config.TopicNames, s.config.PollingPeriod); err != nil {
		return errors.Errorf("could not initialize pubsub client: %w", err)
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
		return opencdc.Record{}, errors.Errorf("failed to get next record: %w", err)
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
	if s.client != nil {
		return s.client.Teardown(ctx)
	}

	return nil
}
