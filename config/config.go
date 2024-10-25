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
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
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

	// Deprecated: use `topicNames` instead.
	TopicName string `json:"topicName"`

	// TopicNames are the TopicNames the source connector will subscribe to
	TopicNames []string `json:"topicNames"`

	// PollingPeriod is the client event polling interval
	PollingPeriod time.Duration `json:"pollingPeriod" default:"100ms"`

	// Replay preset for the position the connector is fetching events from, can be latest or default to earliest.
	ReplayPreset string `json:"replayPreset" default:"earliest"`
}

func (c Config) Validate(ctx context.Context) (Config, error) {
	var errs []error

	if c.TopicName != "" {
		sdk.Logger(ctx).Warn().
			Msg(`"topicName" is deprecated, use "topicNames" instead.`)

		c.TopicNames = slices.Compact(append(c.TopicNames, c.TopicName))
	}

	if len(c.TopicNames) == 0 {
		errs = append(errs, fmt.Errorf("'topicNames' empty, need at least one topic"))
	}

	if c.PollingPeriod == 0 {
		errs = append(errs, fmt.Errorf("polling period cannot be zero %d", c.PollingPeriod))
	}

	if len(errs) != 0 {
		return c, errors.Join(errs...)
	}

	if _, err := url.Parse(c.OAuthEndpoint); err != nil {
		return c, fmt.Errorf("failed to parse oauth endpoint url: %w", err)
	}

	if _, _, err := net.SplitHostPort(c.PubsubAddress); err != nil {
		return c, fmt.Errorf("failed to parse pubsub address: %w", err)
	}

	return c, nil
}
