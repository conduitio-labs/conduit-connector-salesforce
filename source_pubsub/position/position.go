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

func ParseSDKPosition(sdkPos sdk.Position) (Topics, error) {
	var p Topics
	p.Topics = make(TopicPositions)

	if len(sdkPos) == 0 {
		return p, nil
	}

	if err := json.Unmarshal(sdkPos, &p); err != nil {
		return p, fmt.Errorf("invalid position: %w", err)
	}
	return p, nil
}

func NewTopicPosition() Topics {
	var p Topics
	p.Topics = make(TopicPositions)
	return p
}

func (p Topics) SetTopics(topics []string) {
	for _, topic := range topics {
		if _, ok := p.Topics[topic]; !ok {
			replayEvent := TopicPosition{
				ReplayID: nil,
			}
			p.Topics[topic] = replayEvent
		}
	}
}

func (p Topics) GetTopicReplayID(topic string) []byte {
	if p.Topics != nil {
		if _, ok := p.Topics[topic]; ok {
			topicEvent := p.Topics[topic]
			return topicEvent.ReplayID
		}
	}
	return nil
}

func (p Topics) SetTopicReplayID(topic string, replayID []byte) {
	if p.Topics != nil {
		if _, ok := p.Topics[topic]; ok {
			topicEvent := p.Topics[topic]
			topicEvent.ReplayID = replayID
			topicEvent.ReadTime = time.Now()
			p.Topics[topic] = topicEvent
		} else {
			// should never be even reaching this point, something went wrong if we do
			panic(fmt.Errorf("attempting to set replay id - %b on topic %s, topic doesn't exist on position", replayID, topic))
		}
	}
}

func (p Topics) ToSDKPosition() sdk.Position {
	v, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return v
}
