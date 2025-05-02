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

package position

import (
	"encoding/json"
	"time"

	"github.com/conduitio/conduit-commons/opencdc"
	"github.com/go-errors/errors"
)

type Position struct {
	Topics TopicPositions `json:"topics"`
}

type TopicPositions map[string]TopicPosition

type TopicPosition struct {
	ReplayID []byte    `json:"replayID"`
	ReadTime time.Time `json:"readTime"`
}

func ParseSDKPosition(sdkPos opencdc.Position) (*Position, error) {
	if len(sdkPos) == 0 {
		return nil, nil
	}

	var p Position
	if err := json.Unmarshal(sdkPos, &p); err != nil {
		return nil, errors.Errorf("failed to parse topic position: %w", err)
	}

	return &p, nil
}

func New(topics []string) *Position {
	m := make(TopicPositions)

	for _, topic := range topics {
		m[topic] = TopicPosition{}
	}

	return &Position{Topics: m}
}

func (p *Position) WithTopics(topics []string) {
	for _, topic := range topics {
		if _, ok := p.Topics[topic]; !ok {
			p.Topics[topic] = TopicPosition{}
		}
	}
}

func (p *Position) ToSDKPosition() opencdc.Position {
	v, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return v
}
