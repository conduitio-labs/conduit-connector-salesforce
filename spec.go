package salesforce

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
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
			"environment": {
				Default:     "",
				Required:    true,
				Description: "Authorization service based on Organization’s Domain Name (e.g.: https://MyDomainName.my.salesforce.com -> `MyDomainName`) or `sandbox` for test environment.",
			},
			"clientId": {
				Default:     "",
				Required:    true,
				Description: "OAuth Client ID (Consumer Key).",
			},
			"clientSecret": {
				Default:     "",
				Required:    true,
				Description: "OAuth Client Secret (Consumer Secret).",
			},
			"username": {
				Default:     "",
				Required:    true,
				Description: "Username.",
			},
			"password": {
				Default:     "",
				Required:    true,
				Description: "Password.",
			},
			"securityToken": {
				Default:     "",
				Required:    false,
				Description: "Security token as described here: https://help.salesforce.com/s/articleView?id=sf.user_security_token.htm&type=5.",
			},
			"pushTopicName": {
				Default:     "",
				Required:    true,
				Description: "The name or name pattern of the Push Topic to listen to. This value will be prefixed with `/topic/`.",
			},
			"keyField": {
				Default:     "id",
				Required:    true,
				Description: "The name of the data Key field.",
			},
		},
	}
}
