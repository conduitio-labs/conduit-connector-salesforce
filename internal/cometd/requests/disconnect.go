package requests

import (
	"encoding/json"
)

// DisconnectRequest represents disconnection request.
// See: https://docs.cometd.org/current7/reference/#_disconnect_request
type DisconnectRequest struct {
	ClientID string
}

func (r DisconnectRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"channel":  "/meta/disconnect",
		"clientId": r.ClientID,
	})
}
