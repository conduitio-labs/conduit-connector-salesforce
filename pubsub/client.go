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
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	rt "github.com/avast/retry-go/v4"
	config "github.com/conduitio-labs/conduit-connector-salesforce/config"
	eventbusv1 "github.com/conduitio-labs/conduit-connector-salesforce/proto/eventbus/v1"
	"github.com/conduitio-labs/conduit-connector-salesforce/source/position"
	"github.com/conduitio/conduit-commons/opencdc"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/linkedin/goavro/v2"
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

	oauth authenticator

	conn         *grpc.ClientConn
	pubSubClient eventbusv1.PubSubClient

	codecCache  map[string]*goavro.Codec
	unionFields map[string]map[string]struct{}

	buffer chan ConnectResponseEvent

	stop          func()
	tomb          *tomb.Tomb
	topicNames    []string
	currentPos    position.Topics
	maxRetries    uint
	fetchInterval time.Duration
}

type Topic struct {
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

// Creates a new connection to the gRPC server and returns the wrapper struct.
func NewGRPCClient(ctx context.Context, config config.Config, currentPos position.Topics) (*Client, error) {
	sdk.Logger(ctx).Info().
		Strs("topics", config.TopicNames).
		Msgf("Starting GRPC client")

	var transportCreds credentials.TransportCredentials
	var replayPreset eventbusv1.ReplayPreset

	if config.InsecureSkipVerify {
		transportCreds = insecure.NewCredentials()
	} else {
		transportCreds = credentials.NewClientTLSFromCert(getCerts(), "")
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
	}

	conn, err := grpc.NewClient(config.PubsubAddress, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("gRPC dial: %w", err)
	}

	creds, err := NewCredentials(config.ClientID, config.ClientSecret, config.OAuthEndpoint)
	if err != nil {
		return nil, err
	}

	if config.ReplayPreset == "latest" {
		replayPreset = eventbusv1.ReplayPreset_LATEST
	} else {
		replayPreset = eventbusv1.ReplayPreset_EARLIEST
	}

	currentPos.SetTopics(config.TopicNames)

	return &Client{
		conn:          conn,
		pubSubClient:  eventbusv1.NewPubSubClient(conn),
		codecCache:    make(map[string]*goavro.Codec),
		unionFields:   make(map[string]map[string]struct{}),
		replayPreset:  replayPreset,
		buffer:        make(chan ConnectResponseEvent),
		fetchInterval: config.PollingPeriod,
		topicNames:    config.TopicNames,
		currentPos:    currentPos,
		oauth:         &oauth{Credentials: creds},
		maxRetries:    config.RetryCount,
	}, nil
}

// Initializes the pubsub client by authenticating and.
func (c *Client) Initialize(ctx context.Context) error {
	sdk.Logger(ctx).Info().Msgf("Initizalizing PubSub client")

	if err := c.login(ctx); err != nil {
		return err
	}

	if err := c.canSubscribe(ctx); err != nil {
		return err
	}

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

// Wrapper function around the GetTopic RPC. This will add the OAuth credentials and make a call to fetch data about a specific topic.
func (c *Client) canSubscribe(ctx context.Context) error {
	var trailer metadata.MD

	logger := sdk.Logger(ctx).With().Str("at", "client.canSubscribe").Logger()

	for _, topic := range c.topicNames {
		req := &eventbusv1.TopicRequest{
			TopicName: topic,
		}

		ctx, cancel := context.WithTimeout(c.getAuthContext(), GRPCCallTimeout)
		defer cancel()

		resp, err := c.pubSubClient.GetTopic(ctx, req, grpc.Trailer(&trailer))
		logger.Debug().Bool("trailers", true).Fields(trailer).Msg("retrieved topic")
		if err != nil {
			return fmt.Errorf("failed to retrieve topic %q: %w", topic, err)
		}

		if !resp.CanSubscribe {
			return fmt.Errorf("user %q not allowed to subscribe to %q", c.userID, resp.TopicName)
		}

		logger.Debug().
			Bool("can_subscribe", resp.CanSubscribe).
			Str("topic", resp.TopicName).
			Msgf("client allowed to subscribe to events on %q", topic)
	}

	return nil
}

// Next returns the next record from the buffer.
func (c *Client) Next(ctx context.Context) (opencdc.Record, error) {
	select {
	case <-ctx.Done():
		return opencdc.Record{}, fmt.Errorf("next: context done: %w", ctx.Err())
	case event, ok := <-c.buffer:
		if !ok {
			if err := c.tomb.Err(); err != nil {
				return opencdc.Record{}, fmt.Errorf("tomb exited: %w", err)
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

func (c *Client) retryAuth(ctx context.Context, retry bool, topic Topic) (bool, Topic, error) {
	var err error
	sdk.Logger(ctx).Info().Msgf("retry connection on topic %s - retries remaining %d ", topic.topicName, topic.retryCount)
	topic.retryCount--

	if err = c.login(ctx); err != nil && topic.retryCount <= 0 {
		return retry, topic, fmt.Errorf("failed to refresh auth: %w", err)
	} else if err != nil {
		sdk.Logger(ctx).Info().Msgf("received error on login for topic %s - retry - %d : %v ", topic.topicName, topic.retryCount, err)
		retry = true
		return retry, topic, fmt.Errorf("received error on subscribe for topic %s - retry - %d : %w", topic.topicName, topic.retryCount, err)
	}

	if err := c.canSubscribe(ctx); err != nil && topic.retryCount <= 0 {
		return retry, topic, fmt.Errorf("failed to subscribe to client topic %s: %w", topic.topicName, err)
	} else if err != nil {
		sdk.Logger(ctx).Info().Msgf("received error on subscribe for topic %s - retry - %d : %v", topic.topicName, topic.retryCount, err)
		retry = true
		return retry, topic, fmt.Errorf("received error on subscribe for topic %s - retry - %d : %w ", topic.topicName, topic.retryCount, err)
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
				return fmt.Errorf("error retrying (number of retries %d) for topic %s auth - %s", topic.retryCount, topic.topicName, err)
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

				return fmt.Errorf("error recv events: %w", err)
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
		return opencdc.Record{}, fmt.Errorf("err setting replay id %s on an event for topic %s : %w", event.ReplayID, event.Topic, err)
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

// Closes the underlying connection to the gRPC server.
func (c *Client) Close(ctx context.Context) error {
	if c.conn != nil {
		sdk.Logger(ctx).Debug().Msg("closing pubsub gRPC connection")

		return c.conn.Close()
	}

	sdk.Logger(ctx).Debug().Msg("no-op, pubsub connection is not open")

	return nil
}

// Wrapper function around the GetSchema RPC. This will add the OAuth credentials and make a call to fetch data about a specific schema.
func (c *Client) GetSchema(schemaID string) (*eventbusv1.SchemaInfo, error) {
	var trailer metadata.MD

	req := &eventbusv1.SchemaRequest{
		SchemaId: schemaID,
	}

	ctx, cancel := context.WithTimeout(c.getAuthContext(), GRPCCallTimeout)
	defer cancel()

	resp, err := c.pubSubClient.GetSchema(ctx, req, grpc.Trailer(&trailer))
	sdk.Logger(ctx).
		Debug().
		Str("at", "client.getschema").
		Bool("trailers", true).
		Fields(trailer).
		Msg("retrieved schema")
	if err != nil {
		return nil, fmt.Errorf("error getting schema from salesforce api - %s", err)
	}

	return resp, nil
}

// Wrapper function around the Subscribe RPC. This will add the OAuth credentials and create a separate streaming client that will be used to,
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
		return nil, fmt.Errorf("failed to subscribe to topic %q: %w", topic, err)
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
			return nil, fmt.Errorf("received EOF on initial fetch: %w", err)
		}

		return nil, fmt.Errorf("initial fetch request failed: %w", err)
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
		return nil, fmt.Errorf("error subscribing to topic on custom replay id %q: %w",
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
			return nil, fmt.Errorf("pubsub: stream closed when receiving events: %w", err)
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

		codec, err := c.fetchCodec(ctx, e.Event.SchemaId)
		if err != nil {
			return events, err
		}

		parsed, _, err := codec.NativeFromBinary(e.Event.Payload)
		if err != nil {
			return events, err
		}

		payload, ok := parsed.(map[string]interface{})
		if !ok {
			return events, fmt.Errorf("invalid payload type %T", payload)
		}

		logger.Trace().Fields(payload).Msg("decoded event")

		events = append(events, ConnectResponseEvent{
			ReplayID:   e.ReplayId,
			EventID:    e.Event.Id,
			Data:       flattenUnionFields(ctx, payload, c.unionFields[e.Event.SchemaId]),
			Topic:      topic,
			ReceivedAt: time.Now(),
		})
	}

	return events, nil
}

// Unexported helper function to retrieve the cached codec from the PubSubClient's schema cache. If the schema ID is not found in the cache,
// then a GetSchema call is made and the corresponding codec is cached for future use.
func (c *Client) fetchCodec(ctx context.Context, schemaID string) (*goavro.Codec, error) {
	logger := sdk.Logger(ctx).
		With().
		Str("at", "client.fetchcodec").
		Str("schema_id", schemaID).
		Logger()

	codec, ok := c.codecCache[schemaID]
	if ok {
		return codec, nil
	}

	schema, err := c.GetSchema(schemaID)
	if err != nil {
		return nil, fmt.Errorf("error making getschema request for uncached schema - %s", err)
	}

	schemaJSON := schema.GetSchemaJson()

	logger.Trace().Str("schema_json", schemaJSON).Msg("fetched schema")

	codec, err = goavro.NewCodec(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("error creating codec from uncached schema - %s", err)
	}

	c.codecCache[schemaID] = codec

	unionFields, err := parseUnionFields(ctx, schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("error parsing union fields from schema: %w", err)
	}

	c.unionFields[schemaID] = unionFields

	return codec, nil
}

const (
	tokenHeader    = "accesstoken"
	instanceHeader = "instanceurl"
	tenantHeader   = "tenantid"
)

// Returns a new context with the necessary authentication parameters for the gRPC server.
func (c *Client) getAuthContext() context.Context {
	pairs := metadata.Pairs(
		tokenHeader, c.accessToken,
		instanceHeader, c.instanceURL,
		tenantHeader, c.orgID,
	)

	return metadata.NewOutgoingContext(context.Background(), pairs)
}
