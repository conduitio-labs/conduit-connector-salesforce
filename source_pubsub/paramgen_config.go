// Code generated by paramgen. DO NOT EDIT.
// Source: github.com/ConduitIO/conduit-connector-sdk/tree/main/cmd/paramgen

package source

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
)

func (Config) Parameters() map[string]sdk.Parameter {
	return map[string]sdk.Parameter{
		"clientID": {
			Default:     "",
			Description: "clientID is the client id from the salesforce app",
			Type:        sdk.ParameterTypeString,
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
		},
		"clientSecret": {
			Default:     "",
			Description: "clientSecret is the client secret from the salesforce app",
			Type:        sdk.ParameterTypeString,
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
		},
		"insecureSkipVerify": {
			Default:     "",
			Description: "insecureSkipVerify disables certificate validation",
			Type:        sdk.ParameterTypeBool,
			Validations: []sdk.Validation{},
		},
		"oauthEndpoint": {
			Default:     "",
			Description: "oauthEndpoint is the oauthEndpoint from the salesforce app",
			Type:        sdk.ParameterTypeString,
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
		},
		"pollingPeriod": {
			Default:     "100ms",
			Description: "pollingPeriod is the client event polling interval",
			Type:        sdk.ParameterTypeDuration,
			Validations: []sdk.Validation{},
		},
		"pubsubAddress": {
			Default:     "api.pubsub.salesforce.com:7443",
			Description: "gRPC Pubsub Salesforce API address",
			Type:        sdk.ParameterTypeString,
			Validations: []sdk.Validation{},
		},
		"replayPreset": {
			Default:     "earliest",
			Description: "Replay preset for the position the connector is fetching events from, can be latest or default to earliest.",
			Type:        sdk.ParameterTypeString,
			Validations: []sdk.Validation{},
		},
		"retryCount": {
			Default:     "10",
			Description: "Number of retries allowed per read before the connector errors out",
			Type:        sdk.ParameterTypeInt,
			Validations: []sdk.Validation{},
		},
		"topicName": {
			Default:     "",
			Description: "topicName is the topic the source connector will subscribe to",
			Type:        sdk.ParameterTypeString,
			Validations: []sdk.Validation{
				sdk.ValidationRequired{},
			},
		},
		"username": {
			Default:     "",
			Description: "Deprecated: username is the client secret from the salesforce app.",
			Type:        sdk.ParameterTypeString,
			Validations: []sdk.Validation{},
		},
	}
}
