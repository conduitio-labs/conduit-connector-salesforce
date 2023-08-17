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
	// Channel value MUST be `/meta/connect`
	Channel string `json:"channel"`

	// Successful is a boolean indicating the success or failure of the connection
	Successful bool `json:"successful"`

	// Error is a description of the reason for the failure
	Error string `json:"error,omitempty"`

	// Advice is the `advice` object
	Advice *advice `json:"advice,omitempty"`

	// Ext is the `ext` object
	Ext *ext `json:"ext,omitempty"`

	// ClientID is the client ID returned in the handshake response
	ClientID string `json:"clientId,omitempty"`

	// ID is the same value as request message id
	ID string `json:"id,omitempty"`

	// Events is an array of data returned in the response
	Events []ConnectResponseEvent
}

type ConnectResponseEventType = string

const (
	CreatedEventType   = "created"
	UpdatedEventType   = "updated"
	DeletedEventType   = "deleted"
	UndeletedEventType = "undeleted"
)

// ConnectResponseEvent represents single piece of data returned in connect response.
type ConnectResponseEvent struct {
	Data    ConnectResponseEventData `json:"data"`
	Channel string                   `json:"channel"`
}

type ConnectResponseEventData struct {
	Event   ConnectResponseEventDataMetadata `json:"event"`
	SObject map[string]interface{}           `json:"sobject"`
}

type ConnectResponseEventDataMetadata struct {
	CreatedDate time.Time `json:"createdDate"`
	ReplayID    int       `json:"replayId"`

	// Type denotes creation, update, delete, or undelete of a record
	// See: https://developer.salesforce.com/docs/atlas.en-us.236.0.api_streaming.meta/api_streaming/terms.htm
	Type ConnectResponseEventType `json:"type"`
}
