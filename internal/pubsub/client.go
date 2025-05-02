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
	"sync"
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

var ErrEndOfRecords = errors.New("end of records from stream")

type Client struct {
	mu sync.Mutex

	accessToken  string
	instanceURL  string
	userID       string
	orgID        string
	replayPreset eventbusv1.ReplayPreset

	pubSubAction pubSubAction

	oauth Authenticator

	conn         *grpc.ClientConn
	pubSubClient eventbusv1.PubSubClient
	schema       *Schema

	buffer chan ConnectResponseEvent

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

type PublishEvent struct {
	event *eventbusv1.ProducerEvent
	topic Topic
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

const (
	tokenHeader                 = "accesstoken"
	instanceHeader              = "instanceurl"
	tenantHeader                = "tenantid"
	subscribe      pubSubAction = "subscribe"
	publish        pubSubAction = "publish"
)

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

	creds, err := NewCredentials(config.ClientID, config.ClientSecret, config.OAuthEndpoint)
	if err != nil {
		return nil, err
	}

	return &Client{
		schema:       newSchema(c),
		conn:         conn,
		pubSubClient: c,
		buffer:       make(chan ConnectResponseEvent),
		oauth:        &oauth{Credentials: creds},
		maxRetries:   config.RetryCount,
		pubSubAction: pubSubAction(action),
	}, nil
}

// Initializes the pubsub client by authenticating for source and destination.
func (c *Client) Initialize(ctx context.Context, topics []string) error {
	c.topicNames = topics

	if err := c.login(ctx); err != nil {
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
		close(c.buffer)
	}()

	return nil
}

// Next returns the next record from the buffer.
func (c *Client) Next(ctx context.Context) (opencdc.Record, error) {
	select {
	case <-ctx.Done():
		return opencdc.Record{}, errors.Errorf("next: context done: %w", ctx.Err())
	case event, ok := <-c.buffer:
		if !ok {
			if err := c.tomb.Err(); err != nil {
				return opencdc.Record{}, errors.Errorf("tomb exited: %w", err)
			}
			return opencdc.Record{}, ErrEndOfRecords
		}
		return c.buildRecord(event)
	}
}

// Stop ends CDC processing.
func (c *Client) Stop(ctx context.Context) {
	if c.stop != nil {
		c.stop()
	}

	sdk.Logger(ctx).Debug().
		Msgf("stopping pubsub client on topics %q", strings.Join(c.topicNames, ","))

	c.topicNames = nil
}

func (c *Client) Wait(ctx context.Context) error {
	tctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if c.tomb == nil {
		return nil
	}

	select {
	case <-tctx.Done():
		return tctx.Err()
	case <-c.tomb.Dead():
		return c.tomb.Err()
	}
}

// Closes the underlying connection to the gRPC server.
func (c *Client) Close(ctx context.Context) error {
	if c.conn != nil {
		sdk.Logger(ctx).Debug().Msg("closing pubsub gRPC connection")

		return c.conn.Close()
	}

	sdk.Logger(ctx).Debug().Msg("no-op, pubsub connection is not open")

	return nil
}

func (c *Client) GetTopic(topic string) (*eventbusv1.TopicInfo, error) {
	var trailer metadata.MD

	req := &eventbusv1.TopicRequest{
		TopicName: topic,
	}

	ctx, cancel := context.WithTimeout(c.getAuthContext(), GRPCCallTimeout)
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
		topicInfo, err = c.GetTopic(topic.topicName)
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

		err = c.Publish(ctx, &PublishEvent{
			event: event,
			topic: topic,
		})

		return err
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

func (c *Client) Publish(ctx context.Context, publishEvent *PublishEvent) error {
	publishRequest := eventbusv1.PublishRequest{
		TopicName: publishEvent.topic.topicName,
		Events:    []*eventbusv1.ProducerEvent{publishEvent.event},
	}

	sdk.Logger(ctx).Info().
		Str("topic", publishEvent.topic.topicName).
		Str("event id", publishEvent.event.GetId()).
		Msg("Publishing event.")

	resp, err := c.pubSubClient.Publish(c.getAuthContext(), &publishRequest)
	if err != nil {
		return errors.Errorf("error on publishing events: %w", err)
	}

	// check result status on event
	for _, eventRes := range resp.GetResults() {
		if eventRes.GetError() != nil {
			sdk.Logger(ctx).Debug().
				Str("event key", eventRes.GetCorrelationKey()).
				Str("event", eventRes.String()).
				Str("topic_name", publishEvent.topic.topicName).
				Int("number of replays", int(publishEvent.topic.retryCount)). //nolint:gosec //no need to lint retry
				Msgf("failed to publish event: %s", eventRes.GetError())
			return errors.Errorf("failed to publish events %s, retry publish: %s", publishEvent.event.GetId(), eventRes.GetError().GetMsg())
		}
	}

	sdk.Logger(ctx).Info().
		Str("topic", publishEvent.topic.topicName).
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

	subscribeClient, err := c.pubSubClient.Subscribe(c.getAuthContext())
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

func (c *Client) Recv(ctx context.Context, topic string, replayID []byte) ([]ConnectResponseEvent, error) {
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
		if connErr(err) {
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

	// Empty response
	if len(resp.Events) == 0 {
		return []ConnectResponseEvent{}, nil
	}

	logger.Info().
		Str("preset", preset.String()).
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Str("topic", topic).
		Int("events", len(resp.Events)).
		Int32("pending_events", resp.PendingNumRequested).
		Dur("elapsed", time.Since(start)).
		Msg("subscriber received events")

	var events []ConnectResponseEvent

	for _, e := range resp.Events {
		logger.Debug().
			Str("schema_id", e.Event.SchemaId).
			Str("event_id", e.Event.Id).
			Msg("decoding event")

		data, err := c.schema.Unmarshal(ctx, e.Event.SchemaId, e.Event.Payload)
		if err != nil {
			return events, errors.Errorf("failed to unmarshal data: %w", err)
		}

		logger.Trace().Fields(data).Msg("decoded event")

		events = append(events, ConnectResponseEvent{
			ReplayID:   e.ReplayId,
			EventID:    e.Event.Id,
			Data:       data,
			Topic:      topic,
			ReceivedAt: time.Now(),
		})
	}

	return events, nil
}

// Returns a new context with the necessary authentication parameters for the gRPC server.
func (c *Client) getAuthContext() context.Context {
	pairs := metadata.Pairs(
		tokenHeader, c.accessToken,
		instanceHeader, c.instanceURL,
		tenantHeader, c.orgID,
	)

	return metadata.NewOutgoingContext(context.Background(), pairs)
}

func (c *Client) retryAuth(ctx context.Context, retry bool, topic Topic) (bool, Topic, error) {
	var err error
	sdk.Logger(ctx).Info().Msgf("retry connection on topic %s - retries remaining %d ", topic.topicName, topic.retryCount)
	topic.retryCount--

	if err = c.login(ctx); err != nil && topic.retryCount <= 0 {
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

			if len(events) == 0 {
				continue
			}

			sdk.Logger(ctx).Debug().
				Int("events", len(events)).
				Dur("elapsed", time.Since(lastRecvdAt)).
				Str("topic", topic.topicName).
				Msg("received events")

			for _, e := range events {
				topic.replayID = e.ReplayID
				c.buffer <- e
				sdk.Logger(ctx).Debug().
					Int("events", len(events)).
					Dur("elapsed", time.Since(lastRecvdAt)).
					Str("topic", e.Topic).
					Str("replayID", base64.StdEncoding.EncodeToString(e.ReplayID)).
					Msg("record sent to buffer")
			}
		}
	}
}

func (c *Client) buildRecord(event ConnectResponseEvent) (opencdc.Record, error) {
	// TODO - ADD something here to distinguish creates, deletes, updates.
	err := c.currentPos.SetTopicReplayID(event.Topic, event.ReplayID)
	if err != nil {
		return opencdc.Record{}, errors.Errorf("err setting replay id %s on an event for topic %s : %w", event.ReplayID, event.Topic, err)
	}

	sdk.Logger(context.Background()).Debug().
		Dur("elapsed", time.Since(event.ReceivedAt)).
		Str("topic", event.Topic).
		Str("replayID", base64.StdEncoding.EncodeToString(event.ReplayID)).
		Str("replayID uncoded", string(event.ReplayID)).
		Msg("built record, sending it as next")

	return sdk.Util.Source.NewRecordCreate(
		c.currentPos.ToSDKPosition(),
		opencdc.Metadata{
			"opencdc.collection": event.Topic,
		},
		opencdc.StructuredData{
			"replayId": event.ReplayID,
			"id":       event.EventID,
		},
		opencdc.StructuredData(event.Data),
	), nil
}

func (c *Client) login(ctx context.Context) error {
	authResp, err := c.oauth.Login()
	if err != nil {
		return err
	}

	userInfoResp, err := c.oauth.UserInfo(authResp.AccessToken)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.accessToken = authResp.AccessToken
	c.instanceURL = authResp.InstanceURL
	c.userID = userInfoResp.UserID
	c.orgID = userInfoResp.OrganizationID

	sdk.Logger(ctx).Info().
		Str("instance_url", c.instanceURL).
		Str("user_id", c.userID).
		Str("org_id", c.orgID).
		Strs("topics", c.topicNames).
		Msg("successfully authenticated")

	return nil
}

// Wrapper function around the GetTopic RPC. This will add the oauth credentials and make a call to fetch data about a specific topic.
func (c *Client) canAccessTopic(ctx context.Context) error {
	logger := sdk.Logger(ctx).With().Str("at", "client.canAccessTopic").Logger()

	for _, topic := range c.topicNames {
		resp, err := c.GetTopic(topic)
		if err != nil {
			return errors.Errorf(" error retrieving info on topic %s : %s ", topic, err)
		}

		if c.pubSubAction == "subscribe" && !resp.CanSubscribe {
			return errors.Errorf("user %q not allowed to subscribe to %q", c.userID, resp.TopicName)
		}

		if c.pubSubAction == "publish" && !resp.CanPublish {
			return errors.Errorf("user %q not allowed to publish to %q", c.userID, resp.TopicName)
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
func connErr(err error) bool {
	return strings.Contains(err.Error(), "is unavailable")
}

func invalidReplayIDErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "replay id validation failed")
}
