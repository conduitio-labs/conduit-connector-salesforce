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

	"github.com/conduitio-labs/conduit-connector-salesforce/internal/eventbus"
	"github.com/conduitio-labs/conduit-connector-salesforce/source/position"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
)

type Source struct {
	sdk.UnimplementedSource
	config Config

	i *iterator
}

func (s *Source) Config() sdk.SourceConfig {
	return &s.config
}

func NewSource() sdk.Source {
	return sdk.SourceWithMiddleware(&Source{})
}

func (s *Source) Open(ctx context.Context, sdkPos opencdc.Position) error {
	parsedPosition, err := position.ParseSDKPosition(sdkPos)
	if err != nil {
		return errors.Errorf("error parsing sdk position: %w", err)
	}

	c, err := eventbus.NewClient(ctx, s.config.Config)
	if err != nil {
		return errors.Errorf("failed to create eventbus client: %w", err)
	}

	i, err := newIterator(ctx, c, parsedPosition, s.config)
	if err != nil {
		return errors.Errorf("failed to create subscriber iterator: %w", err)
	}

	s.i = i

	sdk.Logger(ctx).Trace().Msg("source open")

	return nil
}

func (s *Source) Read(ctx context.Context) (rec opencdc.Record, err error) {
	sdk.Logger(ctx).Trace().Msg("source read")

	return s.i.Next(ctx)
}

func (s *Source) Ack(ctx context.Context, pos opencdc.Position) error {
	sdk.Logger(ctx).Trace().Hex("position", pos).Msg("source ack")

	return s.i.Ack(ctx, pos)
}

func (s *Source) Teardown(ctx context.Context) error {
	if s.i != nil {
		return s.i.Teardown(ctx)
	}

	sdk.Logger(ctx).Trace().Msg("source teardown")

	return nil
}
