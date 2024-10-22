package destination

//go:generate paramgen -output=paramgen_dest.go DestinationConfig

import (
	"context"
	"fmt"

	pubsub "github.com/conduitio-labs/conduit-connector-salesforce/pubsub"
	"github.com/conduitio/conduit-commons/config"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

var _ client = (*pubsub.Client)(nil)

type client interface {
	Stop(context.Context)
	Close(context.Context) error
	Initialize(context.Context, []string) error
	Publish(context.Context, []opencdc.Record) error
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
	var c Config

	if err := sdk.Util.ParseConfig(
		ctx,
		cfg,
		&c,
		NewDestination().Parameters(),
	); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	c, err := c.Validate(ctx)
	if err != nil {
		return fmt.Errorf("config failed to validate: %w", err)
	}

	d.config = c
	return nil
}

func (d *Destination) Open(ctx context.Context) error {
	logger := sdk.Logger(ctx)

	client, err := pubsub.NewGRPCClient(ctx, d.config.Config)
	if err != nil {
		return fmt.Errorf("could not create GRPCClient: %w", err)
	}

	if err := client.Initialize(ctx, []string{d.config.TopicName}); err != nil {
		return fmt.Errorf("could not initialize pubsub client: %w", err)
	}

	d.client = client

	logger.Debug().
		Str("at", "destination.open").
		Str("topic", d.config.TopicName).
		Msgf("Grpc Client has been set. Will begin read for topic: %s", d.config.TopicName)

	return nil
}

func (d *Destination) Write(ctx context.Context, rr []opencdc.Record) (int, error) {
	if err := d.client.Publish(ctx, rr); err != nil {
		return 0, fmt.Errorf("failed to publish records : %w", err)
	}

	return len(rr), nil
}

func (d *Destination) Teardown(ctx context.Context) error {
	if d.client == nil {
		return nil
	}

	d.client.Stop(ctx)

	if err := d.client.Close(ctx); err != nil {
		return fmt.Errorf("error when closing subscriber conn: %w", err)
	}

	return nil
}
