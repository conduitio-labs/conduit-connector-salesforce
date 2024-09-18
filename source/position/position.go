package position

import (
	"encoding/json"
	"fmt"
	"time"

	sdk "github.com/conduitio/conduit-connector-sdk"
)

type Topics struct {
	Topics TopicPositions `json:"topics"`
}

type TopicPositions map[string]TopicPosition

type TopicPosition struct {
	ReplayID []byte    `json:"replayID"`
	ReadTime time.Time `json:"readTime"`
}

func ParseSDKPosition(sdkPos sdk.Position, topic string) (Topics, error) {
	var p Topics
	p.Topics = make(TopicPositions)

	if len(sdkPos) == 0 {
		return p, nil
	}

	err := json.Unmarshal(sdkPos, &p)
	if err != nil {
		if topic == "" {
			return p, fmt.Errorf("could not parsed sdk position %v: %w", sdkPos, err)
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

func (p Topics) ToSDKPosition() sdk.Position {
	v, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return v
}
