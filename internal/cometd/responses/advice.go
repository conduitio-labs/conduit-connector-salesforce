// Copyright © 2022 Meroxa, Inc. and Miquido
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	// Reconnect indicates how the client should act in the case of a failure to connect.
	// See: https://docs.cometd.org/current7/reference/#_reconnect_advice_field
	Reconnect AdviceReconnect `json:"reconnect,omitempty"`

	// Timeout represents the period of time, in milliseconds, for the server to delay responses to the `/meta/connect` channel.
	// See: https://docs.cometd.org/current7/reference/#_timeout_advice_field
	Timeout int `json:"timeout,omitempty"`

	// Interval represents the minimum period of time, in milliseconds, for a client to delay subsequent requests to the `/meta/connect` channel.
	// A negative period indicates that the message should not be retried.
	// See: https://docs.cometd.org/current7/reference/#_interval_advice_field
	Interval int `json:"interval,omitempty"`

	// MultipleClients is a boolean field, which when true indicates that the server has detected multiple Bayeux client instances running within the same web client.
	// See: https://docs.cometd.org/current7/reference/#_bayeux_multiple_clients_advice
	MultipleClients bool `json:"multiple-clients,omitempty"`

	// Hosts when present, indicates a list of host names or IP addresses that MAY be used as alternate servers with which the client may connect.
	// See: https://docs.cometd.org/current7/reference/#_hosts_advice_field
	Hosts []string `json:"hosts,omitempty"`
}
