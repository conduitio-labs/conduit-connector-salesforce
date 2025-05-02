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

package eventbus

import (
	"bytes"
	"context"
	"io"
	"sync"

	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultFetchSize = int32(10)
)

var errInvalidPositionChange = errors.Errorf("replay id modified")

type TopicConfig struct {
	TopicName    string
	ReplayID     []byte
	ReplayPreset eventbusv1.ReplayPreset
}

type EventData struct {
	ID       string
	Topic    string
	ReplayID []byte
	Data     opencdc.StructuredData
}

type Subscriber struct {
	mu sync.Mutex

	c            *Client
	events       chan *EventData
	sub          eventbusv1.PubSub_SubscribeClient
	topic        string
	lastReplayID []byte
	replayPreset eventbusv1.ReplayPreset
	logger       zerolog.Logger
}

func NewSubscriber(ctx context.Context, c *Client, ch chan *EventData, config TopicConfig) (*Subscriber, error) {
	if err := c.CanSubscribe(ctx, config.TopicName); err != nil {
		return nil, errors.Errorf("unable to subscribe to topic %q: %w", config.TopicName, err)
	}

	sub, err := c.Subscribe(ctx)
	if err != nil {
		return nil, errors.Errorf("failed to create subscriber to %q: %w", config.TopicName, err)
	}

	return &Subscriber{
		c:            c,
		sub:          sub,
		events:       ch,
		topic:        config.TopicName,
		lastReplayID: config.ReplayID,
		replayPreset: config.ReplayPreset,
		logger:       sdk.Logger(ctx).With().Str("topic", config.TopicName).Logger(),
	}, nil
}

func (s *Subscriber) Run(ctx context.Context) error {
	s.logger.Info().Hex("replay_id", s.lastReplayID).Msg("starting subscriber")

	for {
		errChan := make(chan error)

		go func() {
			if err := s.sendRecv(ctx, defaultFetchSize); err != nil {
				s.logger.Error().Err(err).Msg("sendrecv exited with error")
				errChan <- err
			}
			close(errChan)
		}()

		select {
		case err := <-errChan:
			if fatalErr := s.recoverErr(ctx, err); fatalErr != nil {
				return fatalErr
			}
		case <-ctx.Done():
			s.logger.Debug().Msg("context done")
			return ctx.Err()
		}
	}
}

// sendRecv sends request for `n` number of messages and waits until all events are
// received or error is encountered.
// Returns error when:
// * Send/Recv fails.
// * Failed to build the event.
func (s *Subscriber) sendRecv(ctx context.Context, n int32) error {
	logger := s.logger.With().Str("at", "sub.sendRecv").Int32("fetch_size", n).Logger()

	req := &eventbusv1.FetchRequest{
		TopicName:    s.topic,
		ReplayPreset: s.replayPreset,
		NumRequested: n,
	}

	if len(s.lastReplayID) != 0 {
		req.ReplayPreset = eventbusv1.ReplayPreset_CUSTOM
		req.ReplayId = s.lastReplayID
	}

	logger.Debug().Hex("replay_id", s.lastReplayID).Msg("sending fetch request")

	if err := s.sub.Send(req); err != nil {
		if !errors.Is(err, io.EOF) {
			return errors.Errorf("failure to send fetch request: %w", err)
		}
	}

	var (
		pending      = n
		total        = 0
		lastReplayID []byte
	)

	for pending > 0 {
		// short-circut when context is cancelled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// N.B. This may never happen, since sendRecv should have
		//      exited before recovery and reset of the replayID and preset happens.
		if len(lastReplayID) > 0 && !bytes.Equal(lastReplayID, s.lastReplayID) {
			return errInvalidPositionChange
		}

		resp, err := s.sub.Recv()
		if err != nil {
			return errors.Errorf("failed to receive from %q stream: %w", s.topic, err)
		}

		logger.Debug().Int32("pending", pending).Int("events", len(resp.Events)).Msg("processing events")

		for _, e := range resp.Events {
			e, err := s.buildEvent(ctx, e)
			if err != nil {
				return errors.Errorf("failed to build record: %w", err)
			}
			s.events <- e
		}

		total += len(resp.Events)

		logger.Debug().Int32("pending", pending).Int("total", total).Hex("replay_id", resp.LatestReplayId).
			Msg("saving position")

		s.lastReplayID = resp.LatestReplayId
		lastReplayID = resp.LatestReplayId
		pending = resp.PendingNumRequested
	}

	return nil
}

func (s *Subscriber) buildEvent(ctx context.Context, e *eventbusv1.ConsumerEvent) (*EventData, error) {
	data, err := s.c.Unmarshal(ctx, e.Event.SchemaId, e.Event.Payload)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal record: %w", err)
	}

	return &EventData{
		ID:       e.Event.Id,
		Topic:    s.topic,
		ReplayID: e.ReplayId,
		Data:     data,
	}, nil
}

func (s *Subscriber) recoverErr(ctx context.Context, err error) error {
	s.logger.Error().Err(err).Msg("recovering error")
	// return early if there is nothing to recover
	if err == nil {
		return nil
	}

	switch status.Code(err) {
	case codes.Unauthenticated:
		return s.recoverAuth(ctx)
	case codes.InvalidArgument:
		return s.recoverInvalidReplay(ctx)
	}

	return errors.Errorf("unrecoverable error for subscriber: %w", err)
}

func (s *Subscriber) recoverInvalidReplay(ctx context.Context) error {
	s.mu.Lock()

	s.replayPreset = eventbusv1.ReplayPreset_EARLIEST
	s.lastReplayID = nil

	s.mu.Unlock()

	return s.recoverAuth(ctx)
}

func (s *Subscriber) recoverAuth(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCtx, err := s.c.Authorize(ctx)
	if err != nil {
		return errors.Errorf("failed to recover auth: %w", err)
	}

	s.sub, err = s.c.Subscribe(authCtx)
	if err != nil {
		return errors.Errorf("failed to recover stream with fresh auth: %w", err)
	}

	return nil
}
