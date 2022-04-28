package responses

type AdviceReconnect string

const (
	AdviceReconnectRetry     AdviceReconnect = "retry"
	AdviceReconnectHandshake AdviceReconnect = "handshake"
	AdviceReconnectNone      AdviceReconnect = "none"
)

// advice provides a way for servers to inform clients of their preferred mode of client operation so that in conjunction with server-enforced limits, Bayeux implementations can prevent resource exhaustion and inelegant failure modes.
// See: https://docs.cometd.org/current7/reference/#_bayeux_advice
type advice struct {
	// Reconnect https://docs.cometd.org/current7/reference/#_reconnect_advice_field
	Reconnect AdviceReconnect `json:"reconnect,omitempty"`

	// Timeout https://docs.cometd.org/current7/reference/#_timeout_advice_field
	Timeout int `json:"timeout,omitempty"`

	// Interval https://docs.cometd.org/current7/reference/#_interval_advice_field
	Interval int `json:"interval,omitempty"`

	// MultipleClients https://docs.cometd.org/current7/reference/#_bayeux_multiple_clients_advice
	MultipleClients bool `json:"multiple-clients,omitempty"`

	// Hosts https://docs.cometd.org/current7/reference/#_hosts_advice_field
	Hosts []string `json:"hosts,omitempty"`
}
