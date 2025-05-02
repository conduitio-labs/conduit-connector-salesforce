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
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultFetchSize int32 = 10
	defaultPrefetch  int32 = int32(float32(0.1) * float32(defaultFetchSize))
)

var errInvalidPositionChange = errors.Errorf("replay id modified")

var _ subClient = (*client)(nil)

type subClient interface {
	Authorize(context.Context) (context.Context, error)
	Unmarshal(context.Context, string, []byte) (opencdc.StructuredData, error)
	Subscribe(context.Context) (eventbusv1.PubSub_SubscribeClient, error)
	CanSubscribe(context.Context, ...string) error
}

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

	c            subClient
	events       chan *EventData
	sub          eventbusv1.PubSub_SubscribeClient
	topic        string
	lastReplayID []byte
	replayPreset eventbusv1.ReplayPreset
	logger       zerolog.Logger
}

func NewSubscriber(ctx context.Context, c subClient, ch chan *EventData, config TopicConfig) (*Subscriber, error) {
	if err := c.CanSubscribe(ctx, config.TopicName); err != nil {
		return nil, errors.Errorf("unable to subscribe to topic %q: %w", config.TopicName, err)
	}

	sub, err := c.Subscribe(ctx)
	if err != nil {
		return nil, errors.Errorf("failed to subscribe to %q: %w", config.TopicName, err)
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

// Run starts the processing loop of the subscriber.
// Unauthenticated and invalid replay ID errors will be recovered by
// re-authorizing and recreating the subscriber stream.
// Any other errors will cause the subscriber to exit with an error.
func (s *Subscriber) Run(ctx context.Context) error {
	s.logger.Info().Hex("replay_id", s.lastReplayID).Msg("starting subscriber")

	// N.B. This is the recovery loop. When the processing go-routine exits
	//      with errors, there will be an attempt to recover the subscriber.
	for {
		errChan := make(chan error)

		go func() {
			defer close(errChan)
			if err := s.process(ctx); err != nil {
				s.logger.Error().Err(err).Msg("sendrecv exited with error")
				errChan <- err
			}
		}()

		// Recover using the parent context.
		select {
		case err := <-errChan:
			if fatalErr := s.recoverErr(ctx, err); fatalErr != nil {
				return fatalErr
			}
		case <-ctx.Done():
			if ctxErr := s.recoverErr(ctx, ctx.Err()); ctxErr != nil {
				return ctxErr
			}
		}
	}
}

// process performs send/recv on a client gRPC stream. Received events are unmarshaled
// and forwarded over the events channel. Requests for new events are made in 'n' chunks.
// Process will exit with error when:
// * Contex is done/cancelled.
// * Send/recv error.
func (s *Subscriber) process(ctx context.Context) error {
	logger := s.logger.With().Str("send_id", uuid.NewString()).Str("at", "sub.process").Logger()

	var (
		pending = int32(0)
		total   = 0
	)

	lastReplayID := s.lastReplayID

	for {
		// short-circut when context is cancelled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// N.B. This may never happen, since sendRecv should have
		//      exited before recovery when reset of the replayID and preset happens.
		if len(lastReplayID) > 0 && !bytes.Equal(lastReplayID, s.lastReplayID) {
			return errInvalidPositionChange
		}

		// When pending is closing to zero, send request for more events.
		if pending <= defaultPrefetch {
			logger.Info().Hex("replay_id", s.lastReplayID).Msg("sending fetch request")

			req := &eventbusv1.FetchRequest{
				TopicName:    s.topic,
				ReplayPreset: s.replayPreset,
				NumRequested: defaultFetchSize,
			}

			if len(s.lastReplayID) != 0 {
				req.ReplayPreset = eventbusv1.ReplayPreset_CUSTOM
				req.ReplayId = s.lastReplayID
			}

			if err := s.sub.Send(req); err != nil {
				if !errors.Is(err, io.EOF) {
					return errors.Errorf("failure to send fetch request: %w", err)
				}
				logger.Info().Msg("fetch request EOF, continue to recv")
			}
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

			s.lastReplayID = resp.LatestReplayId
			lastReplayID = resp.LatestReplayId
			total++
		}

		// This should contain any extra events requested with prefetch.
		pending = resp.PendingNumRequested

		logger.Info().Int32("pending", pending).Int("total", total).Hex("replay_id", resp.LatestReplayId).
			Msgf("%d processed", len(resp.Events))
	}
}

// buildEvent unmarshals the consumer event with the attached schema id.
// Returns error when unmarshaling fails.
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

// recoverErr attempts to recover the subscriber from the provided error.
// Handles only unavailable, unauthenticated and invalid replay id errors.
// Returns error when recovery fails.
func (s *Subscriber) recoverErr(ctx context.Context, err error) error {
	s.logger.Error().Err(err).Msg("recovering error")
	// return early if there is nothing to recover
	if err == nil {
		return nil
	}

	switch status.Code(err) {
	case codes.Unauthenticated, codes.Unavailable:
		return s.recoverAuth(ctx)
	case codes.InvalidArgument:
		return s.recoverInvalidReplay(ctx)
	}

	return errors.Errorf("unrecoverable error for subscriber: %w", err)
}

// recoverInvalidReplay attempts to recover an invalid replay id error
// by zeroing the cached replay and recreating the stream with fresh auth.
func (s *Subscriber) recoverInvalidReplay(ctx context.Context) error {
	s.mu.Lock()

	s.replayPreset = eventbusv1.ReplayPreset_EARLIEST
	s.lastReplayID = nil

	s.logger.Warn().Msg("invalid replay, resetting to earliest preset")

	s.mu.Unlock()

	return s.recoverAuth(ctx)
}

// recoverAuth attempts to recover auth errors by re-authorizing and
// recreating the stream.
func (s *Subscriber) recoverAuth(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	authCtx, err := s.c.Authorize(ctx)
	if err != nil {
		return errors.Errorf("failed to recover auth: %w", err)
	}

	s.logger.Info().Msg("re-authorization successful")

	s.sub, err = s.c.Subscribe(authCtx)
	if err != nil {
		return errors.Errorf("failed to recover stream with fresh auth: %w", err)
	}

	s.logger.Info().Msgf("established new stream to %q", s.topic)

	return nil
}
