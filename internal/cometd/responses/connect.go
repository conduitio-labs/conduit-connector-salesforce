package responses

import "time"

// ConnectResponse represents connection response.
// See: https://docs.cometd.org/current7/reference/#_connect_response
type ConnectResponse struct {
	Channel    string  `json:"channel"`
	Successful bool    `json:"successful"`
	Error      string  `json:"error,omitempty"`
	Advice     *advice `json:"advice,omitempty"`
	Ext        *ext    `json:"ext,omitempty"`
	ClientId   string  `json:"clientId,omitempty"`
	Id         string  `json:"id,omitempty"`
	Events     []ConnectResponseEvent
}

type ConnectResponseEvent struct {
	Data struct {
		Event struct {
			CreatedDate time.Time `json:"createdDate"`
			ReplayId    int       `json:"replayId"`
			Type        string    `json:"type"`
		} `json:"event"`
		Sobject map[string]interface{} `json:"sobject"`
	} `json:"data"`
	Channel string `json:"channel"`
}
