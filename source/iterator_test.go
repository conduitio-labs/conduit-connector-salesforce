// Copyright © 2025 Meroxa, Inc.
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

import (
	"context"
	"strings"
	"testing"

	"github.com/conduitio-labs/conduit-connector-salesforce/internal/eventbus"
	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/conduitio-labs/conduit-connector-salesforce/source/position"
	"github.com/conduitio/conduit-commons/opencdc"
	"github.com/go-errors/errors"
	"github.com/matryer/is"
	"github.com/stretchr/testify/mock"
)

func TestIterator_New(t *testing.T) {
	ctx := context.TODO()

	t.Run("starts subscribers", func(t *testing.T) {
		is := is.New(t)

		conf := Config{
			TopicNames:   []string{"topic1"},
			ReplayPreset: "latest",
		}

		mockSub := newMockPubSub_SubscribeClient(t)
		mockSub.EXPECT().Send(mock.Anything).Return(errors.New("bad stuff"))

		mockClient := newMockEventbusClient(t)
		mockClient.EXPECT().ParsePreset("latest").Return(eventbusv1.ReplayPreset_EARLIEST, nil).Once()
		mockClient.EXPECT().CanSubscribe(ctx, "topic1").Return(nil).Once()
		mockClient.EXPECT().Subscribe(ctx).Return(mockSub, nil).Once()
		mockClient.EXPECT().Teardown(ctx).Return(nil).Once()

		i, err := newIterator(ctx, mockClient, nil, conf)
		is.NoErr(err)

		_, err = i.Next(ctx)
		is.True(err != nil)
		is.Equal(
			err.Error(),
			`end of events: unrecoverable error for subscriber: failure to send fetch request: bad stuff`,
		)
		is.NoErr(i.Teardown(ctx))
	})
	t.Run("invalid preset", func(t *testing.T) {
		is := is.New(t)

		conf := Config{
			TopicNames:   []string{"topic1"},
			ReplayPreset: "badpreset",
		}

		mockClient := newMockEventbusClient(t)
		mockClient.EXPECT().ParsePreset("badpreset").Return(-1, errors.New("bad preset"))

		_, err := newIterator(ctx, mockClient, nil, conf)
		is.Equal(err.Error(), `failed to start subscribers: invalid preset "badpreset"`)
	})
	t.Run("fail to subscribe", func(t *testing.T) {
		is := is.New(t)

		conf := Config{
			TopicNames:   []string{"topic1"},
			ReplayPreset: "latest",
		}

		mockClient := newMockEventbusClient(t)
		mockClient.EXPECT().ParsePreset("latest").Return(eventbusv1.ReplayPreset_EARLIEST, nil).Once()
		mockClient.EXPECT().CanSubscribe(ctx, "topic1").Return(errors.New("bad sub")).Once()

		_, err := newIterator(ctx, mockClient, nil, conf)
		is.True(strings.Contains(
			err.Error(),
			`unable to subscribe to topic "topic1": bad sub`,
		))
	})
}

func TestIterator_Next(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.TODO())
	testEvent := &eventbus.EventData{
		ID:       "event-1",
		Topic:    "topic1",
		ReplayID: []byte(`replay-id-1`),
		Data:     opencdc.StructuredData{"expect": "message"},
	}

	i := iterator{
		events:       make(chan *eventbus.EventData),
		lastPosition: position.New([]string{"topic1"}),
	}
	go func() {
		defer cancel()
		i.events <- testEvent
	}()

	r, err := i.Next(ctx)
	is.NoErr(err)
	is.True(strings.Contains(
		string(r.Position),
		`{"topics":{"topic1":{"replayID":"cmVwbGF5LWlkLTE=",`),
	) // skip matching the read time
	is.True(r.Payload.Before == nil)
	is.True(r.Payload.After != nil)
	is.Equal(r.Metadata["opencdc.collection"], "topic1")
	is.Equal(r.Payload.After, opencdc.StructuredData{"expect": "message"})

	is.NoErr(i.Ack(ctx, opencdc.Position{}))
	is.NoErr(i.acks.Wait(context.TODO())) // ensure acks are done
}
