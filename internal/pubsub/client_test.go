// Copyright © 2024 Meroxa, Inc.
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

package pubsub

import (
	"context"
	"testing"
	"time"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPubSubClient_Initialize(t *testing.T) {
	mockAuth := newMockAuthenticator(t)
	ctx := context.Background()

	mockAuth.EXPECT().
		Login().
		Return(&LoginResponse{AccessToken: "token", InstanceURL: "instance-url"}, nil)

	mockAuth.EXPECT().
		UserInfo("token").
		Return(
			&UserInfoResponse{UserID: "my-user-id", OrganizationID: "org-id"},
			nil,
		)

	mockPubSubClient := newMockPubSubClient(t)

	mockPubSubClient.EXPECT().GetTopic(
		mock.Anything,
		&eventbusv1.TopicRequest{TopicName: "my-topic"},
		mock.Anything,
	).Return(
		&eventbusv1.TopicInfo{TopicName: "my-topic", CanSubscribe: true, CanPublish: true},
		nil,
	)

	c := &Client{
		oauth:         mockAuth,
		pubSubClient:  mockPubSubClient,
		buffer:        make(chan ConnectResponseEvent),
		topicNames:    []string{"my-topic"},
		fetchInterval: time.Second * 1,
	}

	require.NoError(t, c.Initialize(ctx, c.topicNames))
}
