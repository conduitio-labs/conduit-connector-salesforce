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

// UnsuccessfulHandshakeResponseError represents the handshake failure response.
// See: https://docs.cometd.org/current7/reference/#_unsuccessful_handshake_response
type UnsuccessfulHandshakeResponseError struct {
	// Channel value MUST be `/meta/handshake`
	Channel string `json:"channel"`

	// Successful value is `false`
	Successful bool `json:"successful"`

	// ErrorDetails is a description of the reason for the failure
	ErrorDetails string `json:"error"`

	// SupportedConnectionTypes is a list of connection types supported by the server for the purposes of the connection being negotiated
	SupportedConnectionTypes []string `json:"supportedConnectionTypes,omitempty"`

	// Advice is the `advice` object
	Advice *advice `json:"advice,omitempty"`

	// Version is the version of the protocol that was negotiated
	Version string `json:"version,omitempty"`

	// MinimumVersion defines minimum version of the protocol supported by the server
	MinimumVersion string `json:"minimumVersion,omitempty"`

	// Ext is the `ext` object
	Ext *ext `json:"ext,omitempty"`

	// ID is the same value as request message id
	ID string `json:"id,omitempty"`
}

func (e UnsuccessfulHandshakeResponseError) Error() string {
	return e.ErrorDetails
}
