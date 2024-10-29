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

package destination

import (
	"context"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/proto/eventbus/v1"
	pubsub "github.com/conduitio-labs/conduit-connector-salesforce/pubsub"
	"github.com/conduitio/conduit-commons/config"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
)

var _ client = (*pubsub.Client)(nil)

type client interface {
	Stop(context.Context)
	Close(context.Context) error
	Write(context.Context, []*eventbusv1.ProducerEvent, map[string]*eventbusv1.ProducerEvent) error
	Initialize(context.Context, []string) error
	PrepareEvents(context.Context, []opencdc.Record) ([]*eventbusv1.ProducerEvent, map[string]*eventbusv1.ProducerEvent, error)
}

type Destination struct {
	sdk.UnimplementedDestination
	client client
	config Config
}

func NewDestination() sdk.Destination {
	return sdk.DestinationWithMiddleware(&Destination{}, sdk.DefaultDestinationMiddleware()...)
}

func (d *Destination) Parameters() config.Parameters {
	return d.config.Parameters()
}

func (d *Destination) Configure(ctx context.Context, cfg config.Config) error {
	if err := sdk.Util.ParseConfig(
		ctx,
		cfg,
		&d.config,
		NewDestination().Parameters(),
	); err != nil {
		return errors.Errorf("failed to parse config: %w", err)
	}

	return nil
}

func (d *Destination) Open(ctx context.Context) error {
	logger := sdk.Logger(ctx)

	client, err := pubsub.NewGRPCClient(ctx, d.config.Config, "publish")
	if err != nil {
		return errors.Errorf("could not create GRPCClient: %w", err)
	}

	if err := client.Initialize(ctx, []string{d.config.TopicName}); err != nil {
		return errors.Errorf("could not initialize pubsub client: %w", err)
	}

	d.client = client

	logger.Debug().
		Str("at", "destination.open").
		Str("topic", d.config.TopicName).
		Msgf("Grpc Client has been set. Will begin read for topic: %s", d.config.TopicName)

	return nil
}

func (d *Destination) Write(ctx context.Context, rr []opencdc.Record) (int, error) {
	events, mappedEvents, err := d.client.PrepareEvents(ctx, rr)
	if err != nil {
		return 0, errors.Errorf("failed to prepare records : %w", err)
	}

	if err := d.client.Write(ctx, events, mappedEvents); err != nil {
		return 0, errors.Errorf("failed to write records : %w", err)
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
