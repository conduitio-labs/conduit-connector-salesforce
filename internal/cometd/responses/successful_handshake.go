package responses

// SuccessfulHandshakeResponse represents the handshake success response.
// See: https://docs.cometd.org/current7/reference/#_successful_handshake_response
type SuccessfulHandshakeResponse struct {
	Channel                  string   `json:"channel"`
	Version                  string   `json:"version"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes,omitempty"`
	ClientId                 string   `json:"clientId"`
	Successful               bool     `json:"successful"`
	MinimumVersion           string   `json:"minimumVersion,omitempty"`
	Advice                   *advice  `json:"advice,omitempty"`
	Ext                      *ext     `json:"ext,omitempty"`
	Id                       string   `json:"id,omitempty"`
	AuthSuccessful           bool     `json:"authSuccessful,omitempty"`
}
