package requests

import (
	"encoding/json"
	"fmt"
)

// UnsubscribePushTopicRequest represents unsubscribing to push topic request.
// See: https://docs.cometd.org/current7/reference/#_unsubscribe_request
type UnsubscribePushTopicRequest struct {
	ClientID  string
	PushTopic string
}

func (r UnsubscribePushTopicRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"channel":      "/meta/unsubscribe",
		"clientId":     r.ClientID,
		"subscription": fmt.Sprintf("/topic/%s", r.PushTopic),
	})
}
