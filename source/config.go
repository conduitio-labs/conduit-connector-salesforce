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
	"errors"
	"fmt"
	"slices"
	"time"

	config "github.com/conduitio-labs/conduit-connector-salesforce/config"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Config struct {
	sdk.DefaultSourceMiddleware

	config.Config

	// Deprecated: use `topicNames` instead.
	TopicName string `json:"topicName"`

	// TopicNames are the TopicNames the source connector will subscribe to
	TopicNames []string `json:"topicNames"`

	// PollingPeriod is the client event polling interval
	PollingPeriod time.Duration `json:"pollingPeriod" default:"100ms"`

	// Replay preset for the position the connector is fetching events from, can be latest or default to earliest.
	ReplayPreset string `json:"replayPreset" default:"earliest" validate:"inclusion=latest|earliest"`
}

func (c *Config) Validate(ctx context.Context) error {
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

	return errors.Join(errs...)
}
