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

package destination

import (
	"context"

	"github.com/conduitio-labs/conduit-connector-salesforce/internal/eventbus"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
)

type publisherClient interface {
	Teardown(context.Context) error
	Publish(context.Context, []opencdc.Record) (int, error)
}

type Destination struct {
	sdk.UnimplementedDestination

	publisher publisherClient
	config    Config
}

func (d *Destination) Config() sdk.DestinationConfig {
	return &d.config
}

func NewDestination() sdk.Destination {
	return sdk.DestinationWithMiddleware(&Destination{})
}

func (d *Destination) Open(ctx context.Context) error {
	c, err := eventbus.NewClient(ctx, d.config.Config)
	if err != nil {
		return errors.Errorf("failed to create eventbus client: %w", err)
	}

	p, err := eventbus.NewPublisher(ctx, c, d.config.TopicName)
	if err != nil {
		return errors.Errorf("failed to create publisher: %w", err)
	}

	d.publisher = p

	return nil
}

func (d *Destination) Write(ctx context.Context, rr []opencdc.Record) (int, error) {
	n, err := d.publisher.Publish(ctx, rr)
	if err != nil {
		return n, errors.Errorf("failed to write records: %w", err)
	}

	return n, nil
}

func (d *Destination) Teardown(ctx context.Context) error {
	if d.publisher != nil {
		return d.publisher.Teardown(ctx)
	}

	return nil
}
