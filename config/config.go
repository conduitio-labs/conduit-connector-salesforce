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

package config

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

//go:generate paramgen -output=paramgen_config.go Config
type Config struct {
	// ClientID is the client id from the salesforce app
	ClientID string `json:"clientID" validate:"required"`

	// ClientSecret is the client secret from the salesforce app
	ClientSecret string `json:"clientSecret" validate:"required"`

	// OAuthEndpoint is the OAuthEndpoint from the salesforce app
	OAuthEndpoint string `json:"oauthEndpoint" validate:"required"`

	// gRPC Pubsub Salesforce API address
	PubsubAddress string `json:"pubsubAddress" default:"api.pubsub.salesforce.com:7443"`

	// InsecureSkipVerify disables certificate validation
	InsecureSkipVerify bool `json:"insecureSkipVerify" default:"false"`

	// Number of retries allowed per read before the connector errors out
	RetryCount uint `json:"retryCount" default:"10"`
}

func (c Config) Validate(ctx context.Context) (Config, error) {
	if _, err := url.Parse(c.OAuthEndpoint); err != nil {
		return c, fmt.Errorf("failed to parse oauth endpoint url: %w", err)
	}

	if c.PubsubAddress == "" {
		return c, fmt.Errorf("invalid pubsub address %q", c.PubsubAddress)
	}

	if _, _, err := net.SplitHostPort(c.PubsubAddress); err != nil {
		fmt.Println("address")
		fmt.Println(c.PubsubAddress)
		return c, fmt.Errorf("failed to parse pubsub address: %w", err)
	}

	return c, nil
}
