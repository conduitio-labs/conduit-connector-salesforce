package responses

// DisconnectResponse represents disconnection response.
// See: https://docs.cometd.org/current7/reference/#_disconnect_response
type DisconnectResponse struct {
	Channel    string `json:"channel"`
	Successful bool   `json:"successful"`
	ClientID   string `json:"clientId,omitempty"`
	Error      string `json:"error,omitempty"`
	Ext        *ext   `json:"ext,omitempty"`
	ID         string `json:"id,omitempty"`
}
