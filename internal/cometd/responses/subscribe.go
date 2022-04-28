package responses

import "fmt"

// SubscribeResponse represents subscription response.
// See: https://docs.cometd.org/current7/reference/#_subscribe_response
type SubscribeResponse struct {
	Channel      string      `json:"channel"`
	Successful   bool        `json:"successful"`
	Subscription interface{} `json:"subscription"`
	Error        string      `json:"error,omitempty"`
	Advice       *advice     `json:"advice,omitempty"`
	Ext          *ext        `json:"ext,omitempty"`
	ClientId     string      `json:"clientId,omitempty"`
	Id           string      `json:"id,omitempty"`
}

func (s SubscribeResponse) GetSubscriptions() []string {
	switch subscription := s.Subscription.(type) {
	case string:
		return []string{subscription}

	case []string:
		return subscription
	}

	panic(fmt.Errorf("unexpected subscriptions data: %#v", s.Subscription))
}
