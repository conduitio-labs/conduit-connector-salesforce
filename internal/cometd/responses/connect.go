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

import "time"

// ConnectResponse represents connection response.
// See: https://docs.cometd.org/current7/reference/#_connect_response
type ConnectResponse struct {
	Channel    string  `json:"channel"`
	Successful bool    `json:"successful"`
	Error      string  `json:"error,omitempty"`
	Advice     *advice `json:"advice,omitempty"`
	Ext        *ext    `json:"ext,omitempty"`
	ClientID   string  `json:"clientId,omitempty"`
	ID         string  `json:"id,omitempty"`
	Events     []ConnectResponseEvent
}

type ConnectResponseEvent struct {
	Data struct {
		Event struct {
			CreatedDate time.Time `json:"createdDate"`
			ReplayID    int       `json:"replayId"`
			Type        string    `json:"type"`
		} `json:"event"`
		Sobject map[string]interface{} `json:"sobject"`
	} `json:"data"`
	Channel string `json:"channel"`
}
