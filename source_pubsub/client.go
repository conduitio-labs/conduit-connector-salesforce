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

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/conduitio-labs/conduit-connector-salesforce/source_pubsub/proto"
	sdk "github.com/conduitio/conduit-connector-sdk"
	"github.com/linkedin/goavro/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gopkg.in/tomb.v2"
)

//go:generate mockery --with-expecter --filename pubsub_client_mock.go --name PubSubClient --dir proto --log-level error
//go:generate mockery --with-expecter --filename pubsub_subscribe_client_mock.go --name PubSub_SubscribeClient --dir proto --log-level error

var (
	GRPCDialTimeout = 5 * time.Second
	GRPCCallTimeout = 5 * time.Second
)

var (
	ErrEndOfRecords = errors.New("end of records from stream")
)

type PubSubClient struct {
	mu sync.Mutex

	accessToken  string
	instanceURL  string
	userID       string
	orgID        string
	replayPreset proto.ReplayPreset

	oauth authenticator

	conn         *grpc.ClientConn
	pubSubClient proto.PubSubClient

	codecCache  map[string]*goavro.Codec
	unionFields map[string]map[string]struct{}

	buffer chan sdk.Record
	ticker *time.Ticker

	currReplayID []byte
	tomb         *tomb.Tomb
	topicName    string
}

type ConnectResponseEvent struct {
	Data     map[string]interface{}
	EventID  string
	ReplayID []byte
}

// Creates a new connection to the gRPC server and returns the wrapper struct.
func NewGRPCClient(ctx context.Context, config Config, sdkPos sdk.Position) (*PubSubClient, error) {
	var transportCreds credentials.TransportCredentials
	var replayPreset proto.ReplayPreset

	if config.InsecureSkipVerify {
		transportCreds = insecure.NewCredentials()
	} else {
		transportCreds = credentials.NewClientTLSFromCert(getCerts(), "")
	}

	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(transportCreds),
	}

	dialCtx, cancel := context.WithTimeout(ctx, GRPCDialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, config.PubsubAddress, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("gRPC dial: %w", err)
	}

	creds, err := NewCredentials(config.ClientID, config.ClientSecret, config.OAuthEndpoint)
	if err != nil {
		return nil, err
	}

	t, _ := tomb.WithContext(ctx)

	if config.ReplayPreset == "latest" {
		replayPreset = proto.ReplayPreset_LATEST
	} else {
		replayPreset = proto.ReplayPreset_EARLIEST
	}

	return &PubSubClient{
		conn:         conn,
		pubSubClient: proto.NewPubSubClient(conn),
		codecCache:   make(map[string]*goavro.Codec),
		unionFields:  make(map[string]map[string]struct{}),
		currReplayID: sdkPos,
		replayPreset: replayPreset,
		buffer:       make(chan sdk.Record),
		ticker:       time.NewTicker(config.PollingPeriod),
		topicName:    config.TopicName,
		oauth:        &oauth{Credentials: creds},
		tomb:         t,
	}, nil
}

// Initializes the pubsub client by authenticating and.
func (c *PubSubClient) Initialize(ctx context.Context) error {
	if err := c.login(ctx); err != nil {
		return err
	}

	if err := c.canSubscribe(ctx); err != nil {
		return err
	}

	c.tomb.Go(func() error {
		ctx := c.tomb.Context(nil) //nolint:staticcheck // SA1012 tomb expects nil
		if err := c.startCDC(ctx); err != nil {
			return err
		}
		return nil
	})

	go func() {
		<-c.tomb.Dead()

		sdk.Logger(ctx).Info().Err(c.tomb.Err()).Msgf("tomb died, closing buffer")
		close(c.buffer)
	}()

	return nil
}

func (c *PubSubClient) login(ctx context.Context) error {
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
		Str("topic", c.topicName).
		Msg("successfully authenticated")

	return nil
}

// Wrapper function around the GetTopic RPC. This will add the OAuth credentials and make a call to fetch data about a specific topic.
func (c *PubSubClient) canSubscribe(_ context.Context) error {
	var trailer metadata.MD

	req := &proto.TopicRequest{
		TopicName: c.topicName,
	}

	ctx, cancel := context.WithTimeout(c.getAuthContext(), GRPCCallTimeout)
	defer cancel()

	resp, err := c.pubSubClient.GetTopic(ctx, req, grpc.Trailer(&trailer))
	if err != nil {
		return fmt.Errorf("failed to retrieve topic %q: %w", c.topicName, err)
	}

	if !resp.CanSubscribe {
		return fmt.Errorf("user %q not allowed to subscribe to %q", c.userID, resp.TopicName)
	}

	sdk.Logger(ctx).Debug().
		Bool("can_subscribe", resp.CanSubscribe).
		Str("topic_name", resp.TopicName).
		Msgf("client allowed to subscribe to events on %q", c.topicName)

	return nil
}

// Next returns the next record from the buffer.
func (c *PubSubClient) Next(ctx context.Context) (sdk.Record, error) {
	select {
	case <-ctx.Done():
		return sdk.Record{}, fmt.Errorf("next: context done: %w", ctx.Err())
	case r, ok := <-c.buffer:
		if !ok {
			if err := c.tomb.Err(); err != nil {
				return sdk.Record{}, fmt.Errorf("tomb exited: %w", err)
			}
			return sdk.Record{}, ErrEndOfRecords
		}
		return r, nil
	}
}

// Stop ends CDC processing.
func (c *PubSubClient) Stop(ctx context.Context) {
	c.ticker.Stop()
	_ = c.tomb.Killf("cdc iterator is stopping")

	sdk.Logger(ctx).Debug().
		Msgf("stopping pubsub client on topic %q", c.topicName)
}

func (c *PubSubClient) Wait(ctx context.Context) error {
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

func (c *PubSubClient) ReplayID() []byte {
	return c.currReplayID
}

func (c *PubSubClient) startCDC(ctx context.Context) error {
	sdk.Logger(ctx).Info().Msg("starting CDC processing..")

	var (
		retryAuth   bool
		lastRecvdAt = time.Now().UTC()
	)

	for {
		if retryAuth {
			sdk.Logger(ctx).Info().Msg("refreshing auth token")
			if err := c.login(ctx); err != nil {
				return fmt.Errorf("failed to refresh auth: %w", err)
			}
			retryAuth = false
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.ticker.C: // detect changes every polling period.
			sdk.Logger(ctx).Debug().
				Dur("elapsed", time.Since(lastRecvdAt)).
				Msg("attempting to receive new events")

			lastRecvdAt = time.Now().UTC()

			events, err := c.Recv(ctx)
			if err != nil {
				if c.invalidReplayIDErr(err) {
					sdk.Logger(ctx).Error().Err(err).Msg("replay id is invalid, retrying")

					c.currReplayID = nil
					break
				}

				if c.authErr(err) {
					sdk.Logger(ctx).Error().Err(err).Msg("retrying authentication")

					retryAuth = true
					break
				}

				return fmt.Errorf("error recv events: %w", err)
			}

			sdk.Logger(ctx).Debug().
				Int("events", len(events)).
				Dur("elapsed", time.Since(lastRecvdAt)).
				Msg("received events")

			for _, e := range events {
				c.buffer <- c.buildRecord(e)
				c.currReplayID = e.ReplayID
			}
		}
	}
}

func (*PubSubClient) invalidReplayIDErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "replay id validation failed")
}

func (*PubSubClient) authErr(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "upstream connect error") ||
		strings.Contains(msg, "invalid_grant") ||
		strings.Contains(msg, "disconnect") ||
		strings.Contains(msg, "service is unavailable")
}

func (c *PubSubClient) buildRecord(event ConnectResponseEvent) sdk.Record {
	// TODO - ADD something here to distinguish creates, deletes, updates.
	return sdk.Util.Source.NewRecordCreate(
		sdk.Position(event.ReplayID),
		sdk.Metadata{},
		sdk.StructuredData{
			"replayId": event.ReplayID,
			"id":       event.EventID,
		},
		sdk.StructuredData(event.Data),
	)
}

// Closes the underlying connection to the gRPC server.
func (c *PubSubClient) Close(ctx context.Context) error {
	if c.conn != nil {
		sdk.Logger(ctx).Debug().Msg("closing pubsub gRPC connection")

		return c.conn.Close()
	}

	sdk.Logger(ctx).Debug().Msg("no-op, pubsub connection is not open")

	return nil
}

// Wrapper function around the GetSchema RPC. This will add the OAuth credentials and make a call to fetch data about a specific schema.
func (c *PubSubClient) GetSchema(schemaID string) (*proto.SchemaInfo, error) {
	var trailer metadata.MD

	req := &proto.SchemaRequest{
		SchemaId: schemaID,
	}

	ctx, cancel := context.WithTimeout(c.getAuthContext(), GRPCCallTimeout)
	defer cancel()

	resp, err := c.pubSubClient.GetSchema(ctx, req, grpc.Trailer(&trailer))
	if err != nil {
		return nil, fmt.Errorf("error getting schema from salesforce api - %s", err)
	}

	return resp, nil
}

// Wrapper function around the Subscribe RPC. This will add the OAuth credentials and create a separate streaming client that will be used to,
// fetch data from the topic. This method will continuously consume messages unless an error occurs; if an error does occur then this method will,
// return the last successfully consumed replayID as well as the error message. If no messages were successfully consumed then this method will return,
// the same replayID that it originally received as a parameter.
func (c *PubSubClient) Subscribe(
	ctx context.Context,
	replayPreset proto.ReplayPreset,
	replayID []byte,
) (proto.PubSub_SubscribeClient, error) {
	start := time.Now().UTC()

	subscribeClient, err := c.pubSubClient.Subscribe(c.getAuthContext())
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic %q: %w", c.topicName, err)
	}

	sdk.Logger(ctx).Debug().
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Str("replay_preset", proto.ReplayPreset_name[int32(replayPreset)]).
		Str("topic_name", c.topicName).
		Dur("elapsed", time.Since(start)).
		Msgf("subscribed to %q", c.topicName)

	initialFetchRequest := &proto.FetchRequest{
		TopicName:    c.topicName,
		ReplayPreset: replayPreset,
		NumRequested: 1,
	}

	if replayPreset == proto.ReplayPreset_CUSTOM && len(replayID) > 0 {
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
		Str("replay_preset", proto.ReplayPreset_name[int32(replayPreset)]).
		Str("topic_name", c.topicName).
		Dur("elapsed", time.Since(start)).
		Msg("first request sent")

	return subscribeClient, nil
}

func (c *PubSubClient) Recv(ctx context.Context) ([]ConnectResponseEvent, error) {
	var (
		replayID []byte
		preset   = c.replayPreset
		start    = time.Now().UTC()
	)

	if len(c.currReplayID) > 0 {
		preset = proto.ReplayPreset_CUSTOM
		replayID = c.currReplayID
	}

	sdk.Logger(ctx).Info().
		Str("preset", preset.String()).
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Msg("preparing to subscribe")

	subClient, err := c.Subscribe(ctx, preset, replayID)
	if err != nil {
		return nil, fmt.Errorf("error subscribing to topic on custom replay id %q: %w",
			base64.StdEncoding.EncodeToString(c.currReplayID),
			err,
		)
	}

	sdk.Logger(ctx).Info().
		Str("preset", preset.String()).
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Dur("elapsed", time.Since(start)).
		Msg("preparing to receive events")

	resp, err := subClient.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("pubsub: stream closed when receiving events: %w", err)
		}

		return nil, fmt.Errorf("pubsub: error when recv events: %w", err)
	}

	sdk.Logger(ctx).Info().
		Str("preset", preset.String()).
		Str("replay_id", base64.StdEncoding.EncodeToString(replayID)).
		Int("events", len(resp.Events)).
		Dur("elapsed", time.Since(start)).
		Msg("subscriber received events")

	var events []ConnectResponseEvent

	for _, e := range resp.Events {
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

		events = append(events, ConnectResponseEvent{
			ReplayID: e.ReplayId,
			EventID:  e.Event.Id,
			Data:     flattenUnionFields(ctx, payload, c.unionFields[e.Event.SchemaId]),
		})
	}

	return events, nil
}

// Unexported helper function to retrieve the cached codec from the PubSubClient's schema cache. If the schema ID is not found in the cache,
// then a GetSchema call is made and the corresponding codec is cached for future use.
func (c *PubSubClient) fetchCodec(ctx context.Context, schemaID string) (*goavro.Codec, error) {
	codec, ok := c.codecCache[schemaID]
	if ok {
		sdk.Logger(ctx).Trace().Msg("Fetched cached codec...")
		return codec, nil
	}

	sdk.Logger(ctx).Trace().Msg("Making GetSchema request for uncached schema...")
	schema, err := c.GetSchema(schemaID)
	if err != nil {
		return nil, fmt.Errorf("error making getschema request for uncached schema - %s", err)
	}

	sdk.Logger(ctx).Trace().Msg("Creating codec from uncached schema...")
	schemaJSON := schema.GetSchemaJson()
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
func (c *PubSubClient) getAuthContext() context.Context {
	pairs := metadata.Pairs(
		tokenHeader, c.accessToken,
		instanceHeader, c.instanceURL,
		tenantHeader, c.orgID,
	)

	return metadata.NewOutgoingContext(context.Background(), pairs)
}

// Fetches system certs and returns them if possible. If unable to fetch system certs then an empty cert pool is returned instead.
func getCerts() *x509.CertPool {
	if certs, err := x509.SystemCertPool(); err == nil {
		return certs
	}

	return x509.NewCertPool()
}

// parseUnionFields parses the schema JSON to identify avro union fields.
func parseUnionFields(_ context.Context, schemaJSON string) (map[string]struct{}, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	unionFields := make(map[string]struct{})
	fields := schema["fields"].([]interface{})
	for _, field := range fields {
		f := field.(map[string]interface{})
		fieldType := f["type"]
		if types, ok := fieldType.([]interface{}); ok && len(types) > 1 {
			unionFields[f["name"].(string)] = struct{}{}
		}
	}
	return unionFields, nil
}

// flattenUnionFields flattens union fields decoded from Avro.
func flattenUnionFields(_ context.Context, data map[string]interface{}, unionFields map[string]struct{}) map[string]interface{} {
	flatData := make(map[string]interface{})
	for key, value := range data {
		if _, ok := unionFields[key]; ok { // Check if this field is a union
			if valueMap, ok := value.(map[string]interface{}); ok && len(valueMap) == 1 {
				for _, actualValue := range valueMap {
					flatData[key] = actualValue
					break
				}
			} else {
				flatData[key] = value
			}
		} else {
			flatData[key] = value
		}
	}

	return flatData
}
