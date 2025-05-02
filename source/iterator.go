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
	"time"

	"github.com/conduitio-labs/conduit-connector-salesforce/internal/eventbus"
	"github.com/conduitio-labs/conduit-connector-salesforce/source/position"
	"github.com/conduitio/conduit-commons/csync"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
	"gopkg.in/tomb.v2"
)

const (
	teardownTimeout = 2 * time.Second
)

type eventbusClient = eventbus.Client

type iterator struct {
	events       chan *eventbus.EventData
	t            *tomb.Tomb
	lastPosition *position.Position
	c            eventbusClient
	config       Config
	acks         csync.WaitGroup
}

func newIterator(ctx context.Context, c eventbusClient, pos *position.Position, conf Config) (*iterator, error) {
	t, _ := tomb.WithContext(ctx)

	if pos == nil {
		pos = position.New(conf.TopicNames)
	}

	// backfill any topics which are missing in the position.
	pos.WithTopics(conf.TopicNames)

	i := &iterator{
		t:            t,
		c:            c,
		events:       make(chan *eventbus.EventData),
		lastPosition: pos,
		config:       conf,
	}

	if err := i.startSubscribers(ctx); err != nil {
		return nil, errors.Errorf("failed to start subscribers: %w", err)
	}

	return i, nil
}

func (i *iterator) Next(ctx context.Context) (opencdc.Record, error) {
	select {
	case <-ctx.Done():
		return opencdc.Record{}, errors.Errorf("iterator stopped: %w", ctx.Err())
	case e, ok := <-i.events:
		if !ok { // closed
			if err := i.acks.Wait(ctx); err != nil {
				return opencdc.Record{}, errors.Errorf("failed to wait for acks: %w", err)
			}
			return opencdc.Record{}, errors.Errorf("end of events: %w", i.t.Err())
		}

		i.acks.Add(1)
		return i.buildRecord(ctx, e), nil
	}
}

func (i *iterator) Ack(_ context.Context, _ opencdc.Position) error {
	i.acks.Done()
	return nil
}

func (i *iterator) Teardown(ctx context.Context) error {
	defer func() {
		if i.c != nil {
			_ = i.c.Teardown(ctx)
		}
	}()

	if i.t == nil {
		return nil
	}

	i.t.Kill(errors.Errorf("connector teardown"))

	select {
	case <-time.After(teardownTimeout):
	case <-i.t.Dead():
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-i.t.Dead():
		return nil
	}
}

func (i *iterator) startSubscribers(ctx context.Context) error {
	var subscribers []*eventbus.Subscriber

	replayPreset, err := i.c.ParsePreset(i.config.ReplayPreset)
	if err != nil {
		return errors.Errorf("invalid preset %q", i.config.ReplayPreset)
	}

	for _, topic := range i.config.TopicNames {
		sub, err := eventbus.NewSubscriber(ctx, i.c, i.events, eventbus.TopicConfig{
			TopicName:    topic,
			ReplayID:     i.lastPosition.Topics[topic].ReplayID,
			ReplayPreset: replayPreset,
		})
		if err != nil {
			return errors.Errorf("failed to create subscriber for %q: %w", topic, err)
		}

		subscribers = append(subscribers, sub)
	}

	for _, sub := range subscribers {
		i.t.Go(func() error {
			return sub.Run(i.t.Context(context.TODO()))
		})
	}

	go func() {
		<-i.t.Dead()
		close(i.events)
	}()

	return nil
}

func (i *iterator) buildRecord(ctx context.Context, e *eventbus.EventData) opencdc.Record {
	logger := sdk.Logger(ctx).With().Str("at", "iterator.buildRecord").Str("topic", e.Topic).Logger()

	// update last known position for the specific topic
	i.lastPosition.Topics[e.Topic] = position.TopicPosition{
		ReplayID: e.ReplayID,
		ReadTime: time.Now().UTC(),
	}

	logger.Debug().Hex("replay_id", e.ReplayID).Str("event_id", e.ID).
		Msg("building record for event")

	return sdk.Util.Source.NewRecordCreate(
		i.lastPosition.ToSDKPosition(),
		opencdc.Metadata{
			"opencdc.collection": e.Topic,
		},
		opencdc.StructuredData{
			"replayId": e.ReplayID,
			"id":       e.ID,
		},
		e.Data,
	)
}
