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
	"errors"
	"fmt"
	"regexp"

	config "github.com/conduitio-labs/conduit-connector-salesforce/config"
	"github.com/conduitio/conduit-commons/opencdc"
)

type TopicFn func(opencdc.Record) (string, error)

var (
	topicRegex     = regexp.MustCompile(`^[a-zA-Z0-9._\-]+$`)
	maxTopicLength = 249
)

//go:generate paramgen -output=paramgen_config.go Config
type Config struct {
	config.Config

	// Topic is Salesforce event or topic to write record
	TopicName string `json:"topicName"`
}

func (c Config) Validate(ctx context.Context) (Config, error) {
	var errs []error

	if len(c.TopicName) == 0 {
		errs = append(errs, fmt.Errorf("'topicNames' empty, need at least one topic"))
	}

	if len(errs) != 0 {
		return c, errors.Join(errs...)
	}

	return c, nil
}
