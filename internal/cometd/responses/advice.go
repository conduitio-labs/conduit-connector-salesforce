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
