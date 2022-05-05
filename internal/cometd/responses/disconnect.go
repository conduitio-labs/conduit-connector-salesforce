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

// DisconnectResponse represents disconnection response.
// See: https://docs.cometd.org/current7/reference/#_disconnect_response
type DisconnectResponse struct {
	// Channel value MUST be `/meta/disconnect`
	Channel string `json:"channel"`

	// Successful is a boolean indicating the success or failure of the disconnect request
	Successful bool `json:"successful"`

	// ClientID is the client ID returned in the handshake response
	ClientID string `json:"clientId,omitempty"`

	// Error is a description of the reason for the failure
	Error string `json:"error,omitempty"`

	// Ext is the `ext` object
	Ext *ext `json:"ext,omitempty"`

	// ID is the same value as request message id
	ID string `json:"id,omitempty"`
}
