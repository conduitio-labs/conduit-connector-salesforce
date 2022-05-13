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

package salesforce

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/miquido/conduit-connector-salesforce/source"
)

func Specification() sdk.Specification {
	return sdk.Specification{
		Name:              "salesforce",
		Summary:           "A Salesforce source plugin for Conduit.",
		Version:           "v0.1.0",
		Author:            "Miquido",
		DestinationParams: map[string]sdk.Parameter{
			//
		},
		SourceParams: map[string]sdk.Parameter{
			source.ConfigKeyEnvironment: {
				Default:     "",
				Required:    true,
				Description: "Authorization service based on Organization’s Domain Name (e.g.: https://MyDomainName.my.salesforce.com -> `MyDomainName`) or `sandbox` for test environment.",
			},
			source.ConfigKeyClientID: {
				Default:     "",
				Required:    true,
				Description: "OAuth Client ID (Consumer Key).",
			},
			source.ConfigKeyClientSecret: {
				Default:     "",
				Required:    true,
				Description: "OAuth Client Secret (Consumer Secret).",
			},
			source.ConfigKeyUsername: {
				Default:     "",
				Required:    true,
				Description: "Username.",
			},
			source.ConfigKeyPassword: {
				Default:     "",
				Required:    true,
				Description: "Password.",
			},
			source.ConfigKeySecurityToken: {
				Default:     "",
				Required:    false,
				Description: "Security token as described here: https://help.salesforce.com/s/articleView?id=sf.user_security_token.htm&type=5.",
			},
			source.ConfigKeyPushTopicsNames: {
				Default:     "",
				Required:    true,
				Description: "The name or name pattern of the Push Topic to listen to. This value will be prefixed with `/topic/`.",
			},
			source.ConfigKeyKeyField: {
				Default:     "Id",
				Required:    false,
				Description: "The name of the field that should be used as a Payload's Key. Empty value will set it to `nil`.",
			},
		},
	}
}
