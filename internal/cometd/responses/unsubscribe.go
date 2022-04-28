package responses

import "fmt"

// UnsubscribeResponse represents subscription removal response.
// See: https://docs.cometd.org/current7/reference/#_unsubscribe_response
type UnsubscribeResponse struct {
	Channel      string      `json:"channel"`
	Successful   bool        `json:"successful"`
	Subscription interface{} `json:"subscription"`
	Error        string      `json:"error,omitempty"`
	Advice       *advice     `json:"advice,omitempty"`
	Ext          *ext        `json:"ext,omitempty"`
	ClientID     string      `json:"clientId,omitempty"`
	ID           string      `json:"id,omitempty"`
}

func (s UnsubscribeResponse) GetSubscriptions() []string {
	switch subscription := s.Subscription.(type) {
	case string:
		return []string{subscription}

	case []string:
		return subscription
	}

	panic(fmt.Errorf("unexpected subscriptions data: %#v", s.Subscription))
}
