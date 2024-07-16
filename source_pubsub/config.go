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
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"
)

//go:generate paramgen -output=paramgen_config.go Config

type Config struct {
	// ClientID is the client id from the salesforce app
	ClientID string `json:"clientID" validate:"required"`

	// ClientSecret is the client secret from the salesforce app
	ClientSecret string `json:"clientSecret" validate:"required"`

	// OAuthEndpoint is the OAuthEndpoint from the salesforce app
	OAuthEndpoint string `json:"oauthEndpoint" validate:"required"`

	// TopicName is the topic the source connector will subscribe to
	TopicNames []string `json:"topicNames" validate:"required"`

	// Deprecated: Username is the client secret from the salesforce app.
	Username string `json:"username"`

	// PollingPeriod is the client event polling interval
	PollingPeriod time.Duration `json:"pollingPeriod" default:"100ms"`

	// gRPC Pubsub Salesforce API address
	PubsubAddress string `json:"pubsubAddress" default:"api.pubsub.salesforce.com:7443"`

	// InsecureSkipVerify disables certificate validation
	InsecureSkipVerify bool `json:"insecureSkipVerify"`

	// Replay preset for the position the connector is fetching events from, can be latest or default to earliest.
	ReplayPreset string `json:"replayPreset" default:"earliest"`
	// Number of retries allowed per read before the connector errors out
	RetryCount int `json:"retryCount" default:"10"`
}

func (c Config) Validate() error {
	var errs []error

	if c.ClientID == "" {
		errs = append(errs, fmt.Errorf("invalid client id %q", c.ClientID))
	}

	if c.ClientSecret == "" {
		errs = append(errs, fmt.Errorf("invalid client secret %q", c.ClientSecret))
	}

	if c.OAuthEndpoint == "" {
		errs = append(errs, fmt.Errorf("invalid oauth endpoint %q", c.OAuthEndpoint))
	}

	if c.OAuthEndpoint != "" {
		if _, err := url.Parse(c.OAuthEndpoint); err != nil {
			errs = append(errs, fmt.Errorf("failed to parse oauth endpoint url: %w", err))
		}
	}

	if len(c.TopicNames) == 0 {
		errs = append(errs, fmt.Errorf("invalid topic name %q", c.TopicNames))
	}

	if c.PollingPeriod == 0 {
		errs = append(errs, fmt.Errorf("polling period cannot be zero %d", c.PollingPeriod))
	}

	if c.PubsubAddress == "" {
		errs = append(errs, fmt.Errorf("invalid pubsub address %q", c.OAuthEndpoint))
	}

	if c.PubsubAddress != "" {
		if _, _, err := net.SplitHostPort(c.PubsubAddress); err != nil {
			errs = append(errs, fmt.Errorf("failed to parse pubsub address: %w", err))
		}
	}

	return errors.Join(errs...)
}
