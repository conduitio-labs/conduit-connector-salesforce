// Code generated by paramgen. DO NOT EDIT.
// Source: github.com/ConduitIO/conduit-commons/tree/main/paramgen

package source

import (
	"github.com/conduitio/conduit-commons/config"
)

const (
	ConfigClientID           = "clientID"
	ConfigClientSecret       = "clientSecret"
	ConfigInsecureSkipVerify = "insecureSkipVerify"
	ConfigOauthEndpoint      = "oauthEndpoint"
	ConfigPollingPeriod      = "pollingPeriod"
	ConfigPubsubAddress      = "pubsubAddress"
	ConfigReplayPreset       = "replayPreset"
	ConfigRetryCount         = "retryCount"
	ConfigTopicName          = "topicName"
	ConfigTopicNames         = "topicNames"
	ConfigUsername           = "username"
)

func (Config) Parameters() map[string]config.Parameter {
	return map[string]config.Parameter{
		ConfigClientID: {
			Default:     "",
			Description: "ClientID is the client id from the salesforce app",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{
				config.ValidationRequired{},
			},
		},
		ConfigClientSecret: {
			Default:     "",
			Description: "ClientSecret is the client secret from the salesforce app",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{
				config.ValidationRequired{},
			},
		},
		ConfigInsecureSkipVerify: {
			Default:     "false",
			Description: "InsecureSkipVerify disables certificate validation",
			Type:        config.ParameterTypeBool,
			Validations: []config.Validation{},
		},
		ConfigOauthEndpoint: {
			Default:     "",
			Description: "OAuthEndpoint is the OAuthEndpoint from the salesforce app",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{
				config.ValidationRequired{},
			},
		},
		ConfigPollingPeriod: {
			Default:     "100ms",
			Description: "PollingPeriod is the client event polling interval",
			Type:        config.ParameterTypeDuration,
			Validations: []config.Validation{},
		},
		ConfigPubsubAddress: {
			Default:     "api.pubsub.salesforce.com:7443",
			Description: "gRPC Pubsub Salesforce API address",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{},
		},
		ConfigReplayPreset: {
			Default:     "earliest",
			Description: "Replay preset for the position the connector is fetching events from, can be latest or default to earliest.",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{},
		},
		ConfigRetryCount: {
			Default:     "10",
			Description: "Number of retries allowed per read before the connector errors out",
			Type:        config.ParameterTypeInt,
			Validations: []config.Validation{},
		},
		ConfigTopicName: {
			Default:     "",
			Description: "TopicName {WARN will be deprecated soon} the TopicName the source connector will subscribe to",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{},
		},
		ConfigTopicNames: {
			Default:     "",
			Description: "TopicNames are the TopicNames the source connector will subscribe to",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{
				config.ValidationRequired{},
			},
		},
		ConfigUsername: {
			Default:     "",
			Description: "Deprecated: Username is the client secret from the salesforce app.",
			Type:        config.ParameterTypeString,
			Validations: []config.Validation{},
		},
	}
}