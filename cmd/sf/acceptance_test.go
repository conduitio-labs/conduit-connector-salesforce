package main

import (
	"testing"

	sdk "github.com/conduitio/conduit-connector-sdk"
	sf "github.com/miquido/conduit-connector-salesforce"
	"github.com/miquido/conduit-connector-salesforce/internal/cometd"
	"github.com/miquido/conduit-connector-salesforce/internal/salesforce/oauth"
	sfSource "github.com/miquido/conduit-connector-salesforce/source"
)

type CustomConfigurableAcceptanceTestDriver struct {
	sdk.ConfigurableAcceptanceTestDriver

	streamingClient *streamingClientMock
}

func (d *CustomConfigurableAcceptanceTestDriver) WriteToSource(_ *testing.T, records []sdk.Record) (results []sdk.Record) {
	d.streamingClient.SetResults(records)

	// No destination connector, return wanted records
	for _, record := range records {
		record.Key = nil

		results = append(results, record)
	}

	return
}

func TestAcceptance(t *testing.T) {
	sourceConfig := map[string]string{
		sfSource.ConfigKeyEnvironment:     oauth.EnvironmentSandbox,
		sfSource.ConfigKeyClientID:        "client-id",
		sfSource.ConfigKeyClientSecret:    "client-secret",
		sfSource.ConfigKeyUsername:        "username",
		sfSource.ConfigKeyPassword:        "password",
		sfSource.ConfigKeySecurityToken:   "security-token",
		sfSource.ConfigKeyPushTopicsNames: "MyTopic1,MyTopic2",
	}

	sfSource.OAuthClientFactory = func(_ oauth.Environment, _, _, _, _, _ string) oauth.Client {
		return &oAuthClientMock{}
	}

	streamingClient := &streamingClientMock{}
	sfSource.StreamingClientFactory = func(_, _ string) (cometd.Client, error) {
		return streamingClient, nil
	}

	sdk.AcceptanceTest(t, &CustomConfigurableAcceptanceTestDriver{
		ConfigurableAcceptanceTestDriver: sdk.ConfigurableAcceptanceTestDriver{
			Config: sdk.ConfigurableAcceptanceTestDriverConfig{
				Connector: sdk.Connector{
					NewSpecification: sf.Specification,
					NewSource:        sfSource.NewSource,
					NewDestination:   nil,
				},

				SourceConfig:     sourceConfig,
				GenerateDataType: sdk.GenerateStructuredData,

				Skip: []string{
					"TestAcceptance/TestSource_Open_ResumeAtPositionCDC",
					"TestAcceptance/TestSource_Open_ResumeAtPositionSnapshot",
				},
			},
		},

		streamingClient: streamingClient,
	})
}
