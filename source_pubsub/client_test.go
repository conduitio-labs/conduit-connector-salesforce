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

package source

/*
import (
	"testing"
	"context"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/mock"
	"github.com/conduitio-labs/conduit-connector-salesforce/source_pubsub/mocks"
	"github.com/conduitio-labs/conduit-connector-salesforce/source_pubsub/proto"
)

func TestPubSubClient_Initialize(t *testing.T) {
	mockAuth := newMockAuthenticator(t)

	mockAuth.EXPECT().Login(Credentials{
		ClientID: "some-id",
		ClientSecret: "some-secret",
		OAuthEndpoint: "oauth-endpoint",
	}).Return(&LoginResponse{AccessToken: "token", InstanceURL: "instance-url"}, nil)

	mockAuth.EXPECT().UserInfo("oauth-endpoint", "token").Return(
		&UserInfoResponse{UserID: "my-user-id", OrganizationID: "org-id"},
		nil,
	)

	mockPubSubClient := mocks.NewPubSubClient(t)

	mockPubSubClient.EXPECT().GetTopic(
		mock.Anything,
		&proto.TopicRequest{TopicName: "my-topic"},
		mock.Anything,
	).Return(
		&proto.TopicInfo{TopicName: "my-topic", CanSubscribe: true},
		nil,
	)

	c := &PubSubClient{
		oauth: mockAuth,
		pubSubClient: mockPubSubClient,
	}

	err := c.Initialize(context.Background(), Config{
		ClientID: "some-id",
		ClientSecret: "some-secret",
		OAuthEndpoint: "oauth-endpoint",
		TopicName: "my-topic",
		PollingPeriod: time.Minute*1,
	})
	require.NoError(t, err)
	require.True(t, c.tomb.Alive())

	// reinit again after stop
	c.Stop()
	err = c.Wait(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "cdc iterator is stopping")

	err = c.Initialize(context.Background(), Config{
		ClientID: "some-id",
		ClientSecret: "some-secret",
		OAuthEndpoint: "oauth-endpoint",
		TopicName: "my-topic",
	})
	t.Cleanup(func(){ c.Stop() })
	require.NoError(t, err)
	require.True(t, c.tomb.Alive())
}
*/
