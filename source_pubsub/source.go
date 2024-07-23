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
	"strings"

	"github.com/conduitio-labs/conduit-connector-salesforce/source_pubsub/position"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

//go:generate mockery --with-expecter --name=client --inpackage --log-level error
type client interface {
	Next(context.Context) (sdk.Record, error)
	Initialize(context.Context) error
	Stop(context.Context)
	Close(context.Context) error
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
	var config Config

	if err := sdk.Util.ParseConfig(cfg, &config); err != nil {
		return fmt.Errorf("failed to parse config - %s", err)
	}

	config, err := config.Validate(ctx)
	if err != nil {
		return fmt.Errorf("config failed to validate: %w", err)
	}

	s.config = config

	return nil
}

func (s *Source) Open(ctx context.Context, sdkPos sdk.Position) error {
	logger := sdk.Logger(ctx)
	var parsedPositions position.Topics
	var err error

	logger.Debug().
		Str("at", "source.open").
		Str("position", base64.StdEncoding.EncodeToString(sdkPos)).
		Str("topic", s.config.TopicName).
		Str("topics", strings.Join(s.config.TopicNames, ",")).
		Msg("Open Source Connector")

	parsedPositions, err = position.ParseSDKPosition(sdkPos)

	if err != nil {
		//if using old config and the position isnt parsable
		//assume its in the old position format
		if len(s.config.TopicName) > 0 {
			parsedPositions = position.NewTopicPosition()
			parsedPositions.SetTopics(s.config.TopicNames)
			parsedPositions.SetTopicReplayID(s.config.TopicName, sdkPos)
		} else {
			return fmt.Errorf("could not parsed sdk position %v", sdkPos)
		}
	}

	client, err := NewGRPCClient(ctx, s.config, parsedPositions)
	if err != nil {
		return fmt.Errorf("could not create GRPCClient: %w", err)
	}

	if err := client.Initialize(ctx); err != nil {
		return fmt.Errorf("could not initialize pubsub client: %w", err)
	}

	s.client = client

	for _, t := range s.config.TopicNames {
		p := parsedPositions.GetTopicReplayID(t)
		logger.Debug().
			Str("at", "source.open").
			Str("position", string(p)).
			Str("position encoded", base64.StdEncoding.EncodeToString(p)).
			Str("topic", t).
			Msgf("Grpc Client has been set. Will begin read for topic: %s", t)
	}

	return nil
}

func (s *Source) Read(ctx context.Context) (rec sdk.Record, err error) {
	logger := sdk.Logger(ctx)
	logger.Debug().
		Str("topic", s.config.TopicName).
		Str("topics", strings.Join(s.config.TopicNames, ",")).
		Msg("begin read")

	r, err := s.client.Next(ctx)
	if err != nil {
		sdk.Logger(ctx).Error().Err(err).Msg("next: failed to get next record")
		return sdk.Record{}, err
	}

	// filter out empty record payloads
	if r.Payload.Before == nil && r.Payload.After == nil {
		logger.Error().Str("record", fmt.Sprintf("%+v", r)).
			Msg("backing off, empty record payload detected")

		return sdk.Record{}, sdk.ErrBackoffRetry
	}

	topic, err := r.Metadata.GetCollection()
	if err != nil {
		return sdk.Record{}, err
	}

	logger.Debug().
		Str("at", "source.read").
		Str("position encoded", base64.StdEncoding.EncodeToString(r.Position)).
		Str("position", string(r.Position)).
		Str("record on topic", topic).
		Msg("sending record")

	return r, nil
}

func (s *Source) Ack(ctx context.Context, pos sdk.Position) error {
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
