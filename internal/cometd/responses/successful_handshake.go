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

// SuccessfulHandshakeResponse represents the handshake success response.
// See: https://docs.cometd.org/current7/reference/#_successful_handshake_response
type SuccessfulHandshakeResponse struct {
	// Channel value MUST be `/meta/handshake`
	Channel string `json:"channel"`

	// Version is the version of the protocol that was negotiated
	Version string `json:"version"`

	// SupportedConnectionTypes is a list of connection types supported by the server for the purposes of the connection being negotiated
	SupportedConnectionTypes []string `json:"supportedConnectionTypes,omitempty"`

	// ClientID is a newly generated unique ID string
	ClientID string `json:"clientId"`

	// Successful value is `true`
	Successful bool `json:"successful"`

	// MinimumVersion defines minimum version of the protocol supported by the server
	MinimumVersion string `json:"minimumVersion,omitempty"`

	// Advice is the `advice` object
	Advice *advice `json:"advice,omitempty"`

	// Ext is the `ext` object
	Ext *ext `json:"ext,omitempty"`

	// ID is the same value as request message id
	ID string `json:"id,omitempty"`

	// AuthSuccessful value is `true`.
	// This field MAY be included to support prototype client implementations that required the authSuccessful field
	AuthSuccessful bool `json:"authSuccessful,omitempty"`
}
