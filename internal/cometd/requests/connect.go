package requests

import "encoding/json"

// ConnectRequest represents connection request.
// See: https://docs.cometd.org/current7/reference/#_connect_request
type ConnectRequest struct {
	ClientId string
}

func (r ConnectRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"channel":        "/meta/connect",
		"clientId":       r.ClientId,
		"connectionType": "long-polling",
	})
}
