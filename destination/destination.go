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

	pubsub "github.com/conduitio-labs/conduit-connector-salesforce/internal/pubsub"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
)

var _ client = (*pubsub.Client)(nil)

type client interface {
	Stop(context.Context)
	Close(context.Context) error
	Write(context.Context, opencdc.Record) error
	Initialize(context.Context, []string) error
}

type Destination struct {
	sdk.UnimplementedDestination

	client client
	config Config
}

func (d *Destination) Config() sdk.DestinationConfig {
	return &d.config
}

func NewDestination() sdk.Destination {
	return sdk.DestinationWithMiddleware(&Destination{})
}

func (d *Destination) Open(ctx context.Context) error {
	client, err := pubsub.NewGRPCClient(d.config.Config, "publish")
	if err != nil {
		return errors.Errorf("could not create GRPCClient: %w", err)
	}

	if err := client.Initialize(ctx, []string{d.config.TopicName}); err != nil {
		return errors.Errorf("could not initialize pubsub client: %w", err)
	}

	d.client = client

	sdk.Logger(ctx).Debug().
		Str("at", "destination.open").
		Str("topic", d.config.TopicName).
		Msgf("Grpc Client has been set. Will begin read for topic: %s", d.config.TopicName)

	return nil
}

func (d *Destination) Write(ctx context.Context, rr []opencdc.Record) (int, error) {
	for i, r := range rr {
		if err := d.client.Write(ctx, r); err != nil {
			return i, errors.Errorf("failed to write records: %w", err)
		}
	}

	return len(rr), nil
}

func (d *Destination) Teardown(ctx context.Context) error {
	if d.client == nil {
		return nil
	}

	d.client.Stop(ctx)

	if err := d.client.Close(ctx); err != nil {
		return errors.Errorf("error when closing subscriber conn: %w", err)
	}

	return nil
}
