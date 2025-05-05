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

package pubsub

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"io"
	"strings"
	"time"

	rt "github.com/avast/retry-go/v4"
	config "github.com/conduitio-labs/conduit-connector-salesforce/config"
	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/internal/proto/eventbus/v1"
	"github.com/conduitio-labs/conduit-connector-salesforce/source/position"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/go-errors/errors"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gopkg.in/tomb.v2"
)

var (
	GRPCCallTimeout = 5 * time.Second
	RetryDelay      = 10 * time.Second
)

type authorizer interface {
	// Authorize retrieves new access token.
	Authorize(context.Context) error
	// Context returns context containing authorization credentials.
	Context(context.Context) context.Context
}

var ErrEndOfRecords = errors.New("end of records from stream")

type Client struct {
	replayPreset eventbusv1.ReplayPreset

	pubSubAction pubSubAction

	oauth authorizer

	conn         *grpc.ClientConn
	pubSubClient eventbusv1.PubSubClient
	schema       *SchemaClient

	records chan opencdc.Record

	stop          func()
	tomb          *tomb.Tomb
	topicNames    []string
	currentPos    position.Topics
	maxRetries    uint
	fetchInterval time.Duration
}

type Topic struct {
	// used to retry connection / login errors on topic
	retryCount uint
	topicName  string
	replayID   []byte
}

type ConnectResponseEvent struct {
	Data            map[string]interface{}
	EventID         string
	ReplayID        []byte
	Topic           string
	ReceivedAt      time.Time
	CurrentPosition opencdc.Position
}

type pubSubAction string

// Creates a new connection to the gRPC server and returns the wrapper struct.
func NewGRPCClient(config config.Config, action string) (*Client, error) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCredentials(config.InsecureSkipVerify)),
	}

	conn, err := grpc.NewClient(config.PubsubAddress, dialOpts...)
	if err != nil {
		return nil, errors.Errorf("gRPC dial: %w", err)
	}
	c := eventbusv1.NewPubSubClient(conn)

	oauth, err := newAuthorizer(config.ClientID, config.ClientSecret, config.OAuthEndpoint)
	if err != nil {
		return nil, errors.Errorf("failed to initialize auth: %w", err)
	}

	return &Client{
		schema:       newSchemaClient(c),
		conn:         conn,
		pubSubClient: c,
		records:      make(chan opencdc.Record),
		oauth:        oauth,
		maxRetries:   config.RetryCount,
		pubSubAction: pubSubAction(action),
	}, nil
}

// Initializes the pubsub client by authenticating for source and destination.
func (c *Client) Initialize(ctx context.Context, topics []string) error {
	c.topicNames = topics

	if err := c.oauth.Authorize(ctx); err != nil {
		return err
	}

	if err := c.canAccessTopic(ctx); err != nil {
		return err
	}

	return nil
}

// Start CDC Routine for Source.
func (c *Client) StartCDC(ctx context.Context, replay string, currentPos position.Topics, topics []string, fetch time.Duration) error {
	sdk.Logger(ctx).Info().Msgf("Initizalizing PubSub client for source cdc")

	if replay == "latest" {
		c.replayPreset = eventbusv1.ReplayPreset_LATEST
	} else {
		c.replayPreset = eventbusv1.ReplayPreset_EARLIEST
	}

	// set topics and position on source
	currentPos.SetTopics(topics)
	c.currentPos = currentPos
	// set fetch for ticket
	c.fetchInterval = fetch

	stopCtx, cancel := context.WithCancel(ctx)
	c.stop = cancel

	c.tomb, _ = tomb.WithContext(stopCtx)

	for _, topic := range c.topicNames {
		c.tomb.Go(func() error {
			ctx := c.tomb.Context(nil) //nolint:staticcheck //bomb expects nil when parent context is set.
			return c.startCDC(ctx, Topic{
				topicName:  topic,
				retryCount: c.maxRetries,
				replayID:   c.currentPos.TopicReplayID(topic),
			})
		})
	}

	go func() {
		<-c.tomb.Dead()

		sdk.Logger(ctx).Info().Err(c.tomb.Err()).Msg("tomb died, closing buffer")
		close(c.records)
	}()

	return nil
}

// Next returns the next record from the buffer.
func (c *Client) Next(ctx context.Context) (opencdc.Record, error) {
	select {
	case <-ctx.Done():
		return opencdc.Record{}, errors.Errorf("next: context done: %w", ctx.Err())
	case r, ok := <-c.records:
		if !ok {
			if err := c.tomb.Err(); err != nil {
				return opencdc.Record{}, errors.Errorf("tomb exited: %w", err)
			}
			return opencdc.Record{}, ErrEndOfRecords
		}
		return r, nil
	}
}

func (c *Client) Teardown(ctx context.Context) error {
	// cancel processing loops
	if c.stop != nil {
		c.stop()
	}

	// close gRPC connection
	if c.conn != nil {
		defer c.conn.Close()
	}

	tctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if c.tomb == nil {
		return nil
	}

	// wait for tomb to exit.
	select {
	case <-tctx.Done():
		return tctx.Err()
	case <-c.tomb.Dead():
		return c.tomb.Err()
	}
}

func (c *Client) GetTopic(ctx context.Context, topic string) (*eventbusv1.TopicInfo, error) {
	var trailer metadata.MD

	req := &eventbusv1.TopicRequest{
		TopicName: topic,
	}

	ctx, cancel := context.WithTimeout(c.oauth.Context(ctx), GRPCCallTimeout)
	defer cancel()

	resp, err := c.pubSubClient.GetTopic(ctx, req, grpc.Trailer(&trailer))
	if err != nil {
		return nil, errors.Errorf("failed to retrieve topic %q: %w", topic, err)
	}
	return resp, nil
}

// TODO - refactor this to allow for multi topic support.
// Write attempts to publish event with retry for any pubsub connection errors.
func (c *Client) Write(ctx context.Context, r opencdc.Record) error {
	var (
		err       error
		topicInfo *eventbusv1.TopicInfo
		retry     bool
	)

	if r.Operation == opencdc.OperationDelete {
		return errors.Errorf("delete operation records are not supported")
	}

	topic := Topic{
		topicName:  c.topicNames[0],
		retryCount: c.maxRetries,
	}

	sdk.Logger(ctx).Info().
		Str("topic", topic.topicName).
		Str("replayID", string(c.currentPos.TopicReplayID(topic.topicName))).
		Msg("Starting write.")

	err = rt.Do(func() error {
		recordID := uuid.New()
		topicInfo, err = c.GetTopic(ctx, topic.topicName)
		if err != nil {
			return errors.Errorf("error on publish, cannot retrieve topic %s : %w", c.topicNames[0], err)
		}

		sdk.Logger(ctx).Info().
			Str("topic", topicInfo.TopicName).
			Msg("Started publishing event")

		data, err := c.schema.Marshal(ctx, topicInfo.SchemaId, r.Payload.After.(opencdc.StructuredData))
		if err != nil {
			return errors.Errorf("failed to marshal data to schema %q: %w", topicInfo.SchemaId, err)
		}

		event := &eventbusv1.ProducerEvent{
			SchemaId: topicInfo.SchemaId,
			Payload:  data,
			Id:       recordID.String(),
		}

		if err := c.Publish(ctx, topic.topicName, event); err != nil {
			return errors.Errorf("failed to publish event: %w", err)
		}

		return nil
	},
		rt.Delay(RetryDelay),
		rt.Attempts(topic.retryCount),
		rt.RetryIf(func(err error) bool {
			if !retry || topic.retryCount > 0 {
				return false
			}

			sdk.Logger(ctx).Info().
				Str("topic", topic.topicName).
				Msgf("retrying pubsub publish, record %s: %s", r.Key, err)

			retry, topic, err = c.retryAuth(ctx, retry, topic)
			if err != nil {
				sdk.Logger(ctx).Error().
					Str("topic", topic.topicName).
					Msgf("failed to re-auth to destination: %s", err)
				return false
			}
			return true
		}),
	)
	if err != nil {
		return errors.Errorf("failed to publish events on topic %s : %w", topic.topicName, err)
	}

	return err
}

func (c *Client) Publish(ctx context.Context, topicName string, e *eventbusv1.ProducerEvent) error {
	sdk.Logger(ctx).Info().
		Str("topic", topicName).
		Str("event id", e.GetId()).
		Msg("Publishing event.")

	resp, err := c.pubSubClient.Publish(c.oauth.Context(ctx), &eventbusv1.PublishRequest{
		TopicName: topicName,
		Events:    []*eventbusv1.ProducerEvent{e},
	})
	if err != nil {
		return errors.Errorf("error on publishing events: %w", err)
	}

	// check result status on event
	for _, eventRes := range resp.GetResults() {
		if eventRes.GetError() != nil {
			sdk.Logger(ctx).Debug().
				Str("event key", eventRes.GetCorrelationKey()).
				Str("event", eventRes.String()).
				Str("topic_name", topicName).
				Msgf("failed to publish event: %s", eventRes.GetError())
			return errors.Errorf("failed to publish events %s, retry publish: %s", e.GetId(), eventRes.GetError().GetMsg())
		}
	}

	sdk.Logger(ctx).Info().
		Str("topic", topicName).
		Str("resp", resp.String()).
		Str("resp rpc id", resp.GetRpcId()).
		Msg("Event published")

	return nil
}

// Wrapper function around the Subscribe RPC. This will add the oauth credentials and create a separate streaming client that will be used to,
// fetch data from the topic. This method will continuously consume messages unless an error occurs; if an error does occur then this method will,
// return the last successfully consumed replayID as well as the error message. If no messages were successfully consumed then this method will return,
// the same replayID that it originally received as a parameter.
func (c *Client) Subscribe(
	ctx context.Context,
	replayPreset eventbusv1.ReplayPreset,
	replayID []byte,
	topic string,
) (eventbusv1.PubSub_SubscribeClient, error) {
	start := time.Now().UTC()

	subscribeClient, err := c.pubSubClient.Subscribe(c.oauth.Context(ctx))
	if err != nil {
		sdk.Logger(ctx).Error().Err(err).
			Str("topic", topic).
			Str("replayID", string(replayID)).
			Msg("failed to subscribe to topic")
		return nil, errors.Errorf("failed to subscribe to topic %q: %w", topic, err)
	}

	sdk.Logger(ctx).Debug().
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Str("replay_preset", eventbusv1.ReplayPreset_name[int32(replayPreset)]).
		Str("topic_name", topic).
		Dur("elapsed", time.Since(start)).
		Msgf("subscribed to %q", topic)

	initialFetchRequest := &eventbusv1.FetchRequest{
		TopicName:    topic,
		ReplayPreset: replayPreset,
		NumRequested: 1,
	}

	if replayPreset == eventbusv1.ReplayPreset_CUSTOM && len(replayID) > 0 {
		initialFetchRequest.ReplayId = replayID
	}

	if err := subscribeClient.Send(initialFetchRequest); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.Errorf("received EOF on initial fetch: %w", err)
		}

		return nil, errors.Errorf("initial fetch request failed: %w", err)
	}

	sdk.Logger(ctx).Debug().
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Str("replay_preset", eventbusv1.ReplayPreset_name[int32(replayPreset)]).
		Str("topic_name", topic).
		Dur("elapsed", time.Since(start)).
		Msg("first request sent")

	return subscribeClient, nil
}

func (c *Client) Recv(ctx context.Context, topic string, replayID []byte) ([]*eventbusv1.ConsumerEvent, error) {
	var (
		preset = c.replayPreset
		start  = time.Now().UTC()
	)

	logger := sdk.Logger(ctx).With().Str("at", "client.recv").Logger()

	if len(replayID) > 0 {
		preset = eventbusv1.ReplayPreset_CUSTOM
	}

	logger.Info().
		Str("preset", preset.String()).
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Str("topic", topic).
		Msg("preparing to subscribe")

	subClient, err := c.Subscribe(ctx, preset, replayID, topic)
	if err != nil {
		return nil, errors.Errorf("error subscribing to topic on custom replay id %q: %s",
			base64.StdEncoding.EncodeToString(replayID),
			err,
		)
	}
	defer func() {
		if err := subClient.CloseSend(); err != nil {
			logger.Error().Err(err).Msg("failed to close sub client")
		}
	}()

	logger.Info().
		Str("preset", preset.String()).
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Str("topic", topic).
		Dur("elapsed", time.Since(start)).
		Msg("preparing to receive events")

	resp, err := subClient.Recv()
	if err != nil {
		logger.Error().Fields(subClient.Trailer()).Msg("recv error")

		if errors.Is(err, io.EOF) {
			return nil, errors.Errorf("pubsub: stream closed when receiving events: %w", err)
		}
		if isUnavailableErr(err) {
			logger.Warn().
				Str("preset", preset.String()).
				Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
				Str("topic", topic).
				Dur("elapsed", time.Since(start)).
				Err(err).
				Msg("error while receiving events - retrying to connect")
			return nil, nil
		}
		return nil, err
	}

	return resp.Events, nil
}

func (c *Client) retryAuth(ctx context.Context, retry bool, topic Topic) (bool, Topic, error) {
	var err error
	sdk.Logger(ctx).Info().Msgf("retry connection on topic %s - retries remaining %d ", topic.topicName, topic.retryCount)
	topic.retryCount--

	if err = c.oauth.Authorize(ctx); err != nil && topic.retryCount <= 0 {
		return retry, topic, errors.Errorf("failed to refresh auth: %w", err)
	} else if err != nil {
		sdk.Logger(ctx).Info().Msgf("received error on login for topic %s - retry - %d : %v ", topic.topicName, topic.retryCount, err)
		retry = true
		return retry, topic, errors.Errorf("received error on login for topic %s - retry - %d : %w", topic.topicName, topic.retryCount, err)
	}

	if err := c.canAccessTopic(ctx); err != nil && topic.retryCount <= 0 {
		return retry, topic, errors.Errorf("failed to access  client topic %s: %w", topic.topicName, err)
	} else if err != nil {
		sdk.Logger(ctx).Info().Msgf("received error on access for topic %s - retry - %d : %v", topic.topicName, topic.retryCount, err)
		retry = true
		return retry, topic, errors.Errorf("received error on access for topic %s - retry - %d : %s ", topic.topicName, topic.retryCount, err)
	}

	retry = false
	return retry, topic, nil
}

func (c *Client) startCDC(ctx context.Context, topic Topic) error {
	sdk.Logger(ctx).Info().
		Str("topic", topic.topicName).
		Str("replayID", string(c.currentPos.TopicReplayID(topic.topicName))).
		Msg("starting CDC processing..")

	var (
		retry       bool
		err         error
		lastRecvdAt = time.Now().UTC()
		ticker      = time.NewTicker(c.fetchInterval)
	)

	defer ticker.Stop()

	for {
		sdk.Logger(ctx).Debug().
			Str("topic", topic.topicName).
			Str("replayID", string(topic.replayID)).
			Bool("retry", retry).
			Uint("retry number", topic.retryCount).
			Err(ctx.Err()).
			Msg("cdc loop")

		if retry && ctx.Err() == nil {
			err := rt.Do(func() error {
				retry, topic, err = c.retryAuth(ctx, retry, topic)
				return err
			},
				rt.Delay(RetryDelay),
				rt.Attempts(topic.retryCount),
			)
			if err != nil {
				return errors.Errorf("error retrying (number of retries %d) for topic %s auth - %w", topic.retryCount, topic.topicName, err)
			}
			// once we are done with retries, reset the count
			topic.retryCount = c.maxRetries
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C: // detect changes every polling period.

			sdk.Logger(ctx).Debug().
				Dur("elapsed", time.Since(lastRecvdAt)).
				Str("topic", topic.topicName).
				Str("replayID", base64.StdEncoding.EncodeToString(topic.replayID)).
				Msg("attempting to receive new events")

			lastRecvdAt = time.Now().UTC()

			events, err := c.Recv(ctx, topic.topicName, topic.replayID)
			if err != nil {
				sdk.Logger(ctx).Error().Err(err).
					Str("topic", topic.topicName).
					Str("replayID", string(topic.replayID)).
					Msgf("received error on event receive: %v", err)

				if topic.retryCount > 0 {
					if invalidReplayIDErr(err) {
						sdk.Logger(ctx).Error().Err(err).
							Str("topic", topic.topicName).
							Str("replayID", string(topic.replayID)).
							Msgf("replay id %s is invalid, retrying from preset", string(topic.replayID))
						topic.replayID = nil
					}
					retry = true
					break
				}

				return errors.Errorf("error recv events: %w", err)
			}

			sdk.Logger(ctx).Debug().
				Int("events", len(events)).
				Dur("elapsed", time.Since(lastRecvdAt)).
				Str("topic", topic.topicName).
				Msg("received events")

			if len(events) == 0 {
				continue
			}

			for i, e := range events {
				r, err := c.buildRecord(ctx, topic.topicName, e)
				if err != nil {
					return errors.Errorf("failed to build record: %w", err)
				}
				topic.replayID = e.ReplayId
				c.records <- r

				sdk.Logger(ctx).Debug().
					Int("events", len(events)).
					Int("count", i).
					Dur("elapsed", time.Since(lastRecvdAt)).
					Str("topic", topic.topicName).
					Str("replayID", base64.StdEncoding.EncodeToString(e.ReplayId)).
					Msg("record sent to buffer")
			}
		}
	}
}

func (c *Client) buildRecord(ctx context.Context, topic string, e *eventbusv1.ConsumerEvent) (opencdc.Record, error) {
	if err := c.currentPos.SetTopicReplayID(topic, e.ReplayId); err != nil {
		return opencdc.Record{}, errors.Errorf("failed to set topic %q replay id: %w", topic, err)
	}

	sdk.Logger(context.Background()).Debug().
		Str("topic", topic).
		Str("replayID", base64.StdEncoding.EncodeToString(e.ReplayId)).
		Msg("built record, sending it as next")

	data, err := c.schema.Unmarshal(ctx, e.Event.SchemaId, e.Event.Payload)
	if err != nil {
		return opencdc.Record{}, errors.Errorf("failed to unmarshal data: %w", err)
	}

	key := opencdc.StructuredData{
		"replayId": e.ReplayId,
		"id":       e.Event.Id,
	}
	meta := opencdc.Metadata{}
	meta.SetCollection(topic)

	return sdk.Util.Source.NewRecordCreate(
		c.currentPos.ToSDKPosition(),
		meta,
		key,
		opencdc.StructuredData(data),
	), nil
}

// Wrapper function around the GetTopic RPC. This will add the oauth credentials and make a call to fetch data about a specific topic.
func (c *Client) canAccessTopic(ctx context.Context) error {
	logger := sdk.Logger(ctx).With().Str("at", "client.canAccessTopic").Logger()

	for _, topic := range c.topicNames {
		resp, err := c.GetTopic(ctx, topic)
		if err != nil {
			return errors.Errorf(" error retrieving info on topic %s : %s ", topic, err)
		}

		if c.pubSubAction == "subscribe" && !resp.CanSubscribe {
			return errors.Errorf("not allowed to subscribe to %q", resp.TopicName)
		}

		if c.pubSubAction == "publish" && !resp.CanPublish {
			return errors.Errorf("not allowed to publish to %q", resp.TopicName)
		}

		logger.Debug().
			Str("topic", resp.TopicName).
			Msgf("client allowed to %s to events on %q", c.pubSubAction, topic)
	}

	return nil
}

func transportCredentials(skipVerify bool) credentials.TransportCredentials {
	if skipVerify {
		return insecure.NewCredentials()
	}

	if pool, err := x509.SystemCertPool(); err == nil {
		return credentials.NewClientTLSFromCert(pool, "")
	}

	return credentials.NewClientTLSFromCert(x509.NewCertPool(), "")
}

// checks connection error.
func isUnavailableErr(err error) bool {
	return strings.Contains(err.Error(), "is unavailable")
}

func invalidReplayIDErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "replay id validation failed")
}
