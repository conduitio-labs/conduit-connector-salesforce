package requests

import "encoding/json"

// HandshakeRequest represents handshake request.
// See: https://docs.cometd.org/current7/reference/#_handshake_request
type HandshakeRequest struct {
}

func (r HandshakeRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"version":        "1.0",
		"minimumVersion": "1.0",
		"channel":        "/meta/handshake",
		"supportedConnectionTypes": []string{
			"long-polling",
		},
	})
}
