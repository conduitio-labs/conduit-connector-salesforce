// Copyright © 2022 Meroxa, Inc.
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

package main

import (
	"testing"

	sf "github.com/conduitio-labs/conduit-connector-salesforce"
	"github.com/conduitio-labs/conduit-connector-salesforce/internal/cometd"
	"github.com/conduitio-labs/conduit-connector-salesforce/internal/salesforce/oauth"
	sfSource "github.com/conduitio-labs/conduit-connector-salesforce/source"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
)

type CustomConfigurableAcceptanceTestDriver struct {
	sdk.ConfigurableAcceptanceTestDriver

	streamingClient *streamingClientMock
}

func (d *CustomConfigurableAcceptanceTestDriver) WriteToSource(_ *testing.T, records []opencdc.Record) (results []opencdc.Record) {
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
