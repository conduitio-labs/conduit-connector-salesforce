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
	Channel                  string   `json:"channel"`
	Version                  string   `json:"version"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes,omitempty"`
	ClientID                 string   `json:"clientId"`
	Successful               bool     `json:"successful"`
	MinimumVersion           string   `json:"minimumVersion,omitempty"`
	Advice                   *advice  `json:"advice,omitempty"`
	Ext                      *ext     `json:"ext,omitempty"`
	ID                       string   `json:"id,omitempty"`
	AuthSuccessful           bool     `json:"authSuccessful,omitempty"`
}
