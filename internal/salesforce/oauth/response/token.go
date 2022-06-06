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

package response

// TokenResponse represents successful OAuth server response.
// See: https://developer.salesforce.com/docs/atlas.en-us.api_iot.meta/api_iot/qs_auth_access_token.htm
type TokenResponse struct {
	// [required] AccessToken contains the access token value. API clients pass the access token in the Authorization
	// header of each request.
	AccessToken string `json:"access_token"`

	// [required] InstanceURL contains the Salesforce instance URL for API calls.
	InstanceURL string `json:"instance_url"`

	ID        string `json:"id,omitempty"`
	IssuedAt  string `json:"issued_at,omitempty"`
	Signature string `json:"signature,omitempty"`
}
