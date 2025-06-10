package pubsub

import (
	"context"
	"crypto/x509"
	"encoding/json"

	"github.com/conduitio-labs/conduit-connector-salesforce/config"
	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type dialer struct {
	oauth  authorizer
	config config.Config
}

func newDialer(conf config.Config, auther authorizer) *dialer {
	return &dialer{
		oauth:  auther,
		config: conf,
	}
}

func (d *dialer) DialOpts() []grpc.DialOption {
	return []grpc.DialOption{
		d.WithServiceConfig(),
		d.WithTransportCredentials(),
		d.WithStreamInterceptor(),
		d.WithUnaryInterceptor(),
	}
}

func (d *dialer) WithStreamInterceptor() grpc.DialOption {
	return grpc.WithStreamInterceptor(d.streamInterceptor)
}

func (d *dialer) WithUnaryInterceptor() grpc.DialOption {
	return grpc.WithUnaryInterceptor(d.unaryInterceptor)
}

func (d *dialer) WithServiceConfig() grpc.DialOption {
	serviceConfig := map[string]any{
		"methodConfig": []map[string]any{
			{
				"name": []map[string]any{
					{
						"service": eventbusv1.PubSub_ServiceDesc.ServiceName,
					},
				},
				"retryPolicy": map[string]any{
					"MaxAttempts":       d.config.RetryCount,
					"InitialBackoff":    ".1s",
					"MaxBackoff":        ".1s",
					"BackoffMultiplier": 1.0,
					"RetryableStatusCodes": []codes.Code{
						codes.Unavailable,
					},
				},
			},
		},
	}

	v, err := json.MarshalIndent(serviceConfig, "", "  ")
	if err != nil {
		panic("failed to marshal gRPC service config: " + err.Error())
	}

	return grpc.WithDefaultServiceConfig(string(v))
}

func (d *dialer) WithTransportCredentials() grpc.DialOption {
	if d.config.InsecureSkipVerify {
		return grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		return grpc.WithTransportCredentials(
			credentials.NewClientTLSFromCert(x509.NewCertPool(), ""),
		)
	}

	return grpc.WithTransportCredentials(
		credentials.NewClientTLSFromCert(pool, ""),
	)
}

func (d *dialer) streamInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	logger := sdk.Logger(ctx).With().Str("grpc_interceptor", "stream").Logger()

	stream, err := streamer(ctx, desc, cc, method, opts...)
	if err == nil {
		return stream, nil
	}

	if status.Code(err) != codes.Unauthenticated {
		return stream, err
	}

	logger.Error().Err(err).Msg("authentication error, re-authorizing..")

	if err := d.oauth.Authorize(ctx); err != nil {
		return nil, errors.Errorf("failed to authorize client: %w", err)
	}

	logger.Info().Msg("authorized, retrying request with new auth data")

	stream, err = streamer(d.oauth.Context(ctx), desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (d *dialer) unaryInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	logger := sdk.Logger(ctx).With().Str("grpc_interceptor", "invoke").Logger()

	err := invoker(ctx, method, req, reply, cc, opts...)
	if err == nil {
		return nil
	}

	if status.Code(err) != codes.Unauthenticated {
		return err
	}

	logger.Error().Err(err).Msg("authentication error, re-authorizing..")

	if err := d.oauth.Authorize(ctx); err != nil {
		return errors.Errorf("failed to authorize client: %w", err)
	}
	logger.Info().Msg("authorized, retrying request with new auth data")

	if err := invoker(d.oauth.Context(ctx), method, req, reply, cc, opts...); err != nil {
		return err
	}

	return nil
}
