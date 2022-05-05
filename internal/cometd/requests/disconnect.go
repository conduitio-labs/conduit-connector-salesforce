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

package requests

import (
	"encoding/json"
)

// DisconnectRequest represents disconnection request.
// See: https://docs.cometd.org/current7/reference/#_disconnect_request
type DisconnectRequest struct {
	// ClientID is the client ID returned in the handshake response
	ClientID string
}

func (r DisconnectRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"channel":  "/meta/disconnect",
		"clientId": r.ClientID,
	})
}
