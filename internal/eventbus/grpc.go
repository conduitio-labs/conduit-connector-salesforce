// Copyright © 2025 Meroxa, Inc.
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

package eventbus

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

type grpcClientExt struct {
	oauth  authorizer
	config config.Config
}

func newClientExt(conf config.Config, auther authorizer) *grpcClientExt {
	return &grpcClientExt{
		oauth:  auther,
		config: conf,
	}
}

func (c *grpcClientExt) DialOpts() []grpc.DialOption {
	return []grpc.DialOption{
		c.WithServiceConfig(),
		c.WithTransportCredentials(),
		c.WithStreamInterceptor(),
		c.WithUnaryInterceptor(),
	}
}

// WithStreamInterceptor returns a stream interceptor dial option.
func (c *grpcClientExt) WithStreamInterceptor() grpc.DialOption {
	return grpc.WithStreamInterceptor(c.streamInterceptor)
}

// WithUnaryInterceptor returns a unary interceptor dial option.
func (c *grpcClientExt) WithUnaryInterceptor() grpc.DialOption {
	return grpc.WithUnaryInterceptor(c.unaryInterceptor)
}

// WithServiceConfig returns a gRPC service config dial option which
// contains default retry policy for the eventbus service.
func (c *grpcClientExt) WithServiceConfig() grpc.DialOption {
	serviceConfig := map[string]any{
		"methodConfig": []map[string]any{
			{
				"name": []map[string]any{
					{
						"service": eventbusv1.PubSub_ServiceDesc.ServiceName,
					},
				},
				"retryPolicy": map[string]any{
					"MaxAttempts":       c.config.RetryCount,
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

// WithTransportCredentials returns gRPC credentials dial option.
// Insecure credentials are returned when verification is disabled.
// Default system cert pool is used when available.
func (c *grpcClientExt) WithTransportCredentials() grpc.DialOption {
	if c.config.InsecureSkipVerify {
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

// streamInterceptor will retry all stream requests failing with auth errors.
// On successful authorization, new auth context will be injected in the retry.
// Returns error when:
// * original error is not auth related.
// * re-authorization fails.
func (c *grpcClientExt) streamInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	logger := sdk.Logger(ctx).With().Str("grpc_interceptor", "stream").Logger()

	stream, err := streamer(c.oauth.Context(ctx), desc, cc, method, opts...)
	if err == nil {
		return stream, nil
	}

	if status.Code(err) != codes.Unauthenticated {
		return stream, err
	}

	logger.Error().Err(err).Msg("authentication error, re-authorizing..")

	if err := c.oauth.Authorize(ctx); err != nil {
		return nil, errors.Errorf("failed to authorize client: %w", err)
	}

	logger.Info().Msg("authorized, retrying request with new auth data")

	stream, err = streamer(c.oauth.Context(ctx), desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

// unaryInterceptor will retry all unary requests requests failing with auth errors.
// On successful authorization, new auth context will be injected in the retry.
// Returns error when:
// * original error is not auth related.
// * re-authorization fails.
func (c *grpcClientExt) unaryInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	logger := sdk.Logger(ctx).With().Str("grpc_interceptor", "invoke").Logger()

	err := invoker(c.oauth.Context(ctx), method, req, reply, cc, opts...)
	if err == nil {
		return nil
	}

	if status.Code(err) != codes.Unauthenticated {
		return err
	}

	logger.Error().Err(err).Msg("authentication error, re-authorizing..")

	if err := c.oauth.Authorize(ctx); err != nil {
		return errors.Errorf("failed to authorize client: %w", err)
	}
	logger.Info().Msg("authorized, retrying request with new auth data")

	if err := invoker(c.oauth.Context(ctx), method, req, reply, cc, opts...); err != nil {
		return err
	}

	return nil
}
