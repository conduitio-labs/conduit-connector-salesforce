package requests

import (
	"encoding/json"
	"fmt"
)

// SubscribePushTopicRequest represents subscribing to push topic request.
// See: https://docs.cometd.org/current7/reference/#_subscribe_request
type SubscribePushTopicRequest struct {
	ClientID  string
	PushTopic string
}

func (r SubscribePushTopicRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"channel":      "/meta/subscribe",
		"clientId":     r.ClientID,
		"subscription": fmt.Sprintf("/topic/%s", r.PushTopic),
	})
}
