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
	"fmt"
	"time"

	"github.com/conduitio/conduit-commons/opencdc"
)

type Topics struct {
	Topics TopicPositions `json:"topics"`
}

type TopicPositions map[string]TopicPosition

type TopicPosition struct {
	ReplayID []byte    `json:"replayID"`
	ReadTime time.Time `json:"readTime"`
}

func ParseSDKPosition(sdkPos opencdc.Position, topic string) (Topics, error) {
	var p Topics
	p.Topics = make(TopicPositions)

	if len(sdkPos) == 0 {
		return p, nil
	}

	err := json.Unmarshal(sdkPos, &p)
	if err != nil {
		if topic == "" {
			return p, fmt.Errorf("could not parse sdk position %v: %w", sdkPos, err)
		}

		p.SetTopics([]string{topic})
		err := p.SetTopicReplayID(topic, sdkPos)
		return p, err
	}

	return p, err
}

func NewTopicPosition() Topics {
	var p Topics
	p.Topics = make(TopicPositions)
	return p
}

func (p Topics) SetTopics(topics []string) {
	for _, topic := range topics {
		if _, ok := p.Topics[topic]; !ok {
			p.Topics[topic] = TopicPosition{
				ReplayID: nil,
			}
		}
	}
}

func (p Topics) TopicReplayID(topic string) []byte {
	if p.Topics != nil {
		if _, ok := p.Topics[topic]; ok {
			return p.Topics[topic].ReplayID
		}
	}
	return nil
}

func (p Topics) SetTopicReplayID(topic string, replayID []byte) error {
	if p.Topics != nil {
		if _, ok := p.Topics[topic]; ok {
			p.Topics[topic] = TopicPosition{
				ReplayID: replayID,
				ReadTime: time.Now(),
			}
		} else {
			// should never be even reaching this point, something went wrong if we do
			return fmt.Errorf("attempting to set replay id - %b on topic %s, topic doesn't exist on position", replayID, topic)
		}
	}
	return nil
}

func (p Topics) ToSDKPosition() opencdc.Position {
	v, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return v
}
