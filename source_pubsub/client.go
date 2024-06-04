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
	// topic and subscription-related variables.

	// gRPC server variables.
	GRPCEndpoint    = "api.pubsub.salesforce.com:7443"
	GRPCDialTimeout = 5 * time.Second
	GRPCCallTimeout = 5 * time.Second
)

var (
	ErrEndOfRecords = errors.New("end of records from stream")
)

type PubSubClient struct {
	mu sync.Mutex

	accessToken string
	instanceURL string
	userID      string
	orgID       string

	oauth authenticator

	conn         *grpc.ClientConn
	pubSubClient proto.PubSubClient
	subClient    proto.PubSub_SubscribeClient

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
	eventID  string
	replayID []byte
}

// Creates a new connection to the gRPC server and returns the wrapper struct.
func NewGRPCClient(ctx context.Context, config Config, sdkPos sdk.Position) (*PubSubClient, error) {
	logger := sdk.Logger(ctx)

	logger.Debug().Msg("NewGRPCClient - Starting GRPC Client")

	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
	}

	if GRPCEndpoint == "localhost:7011" {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		certs := getCerts()
		creds := credentials.NewClientTLSFromCert(certs, "")
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	}

	dialCtx, cancel := context.WithTimeout(ctx, GRPCDialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, GRPCEndpoint, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("error on dialing grpc - %s", err)
	}

	creds, err := NewCredentials(config.ClientID, config.ClientSecret, config.OAuthEndpoint)
	if err != nil {
		return nil, err
	}

	return &PubSubClient{
		conn:         conn,
		pubSubClient: proto.NewPubSubClient(conn),
		codecCache:   make(map[string]*goavro.Codec),
		unionFields:  make(map[string]map[string]struct{}),
		currReplayID: sdkPos,
		buffer:       make(chan sdk.Record),
		ticker:       time.NewTicker(config.PollingPeriod),
		topicName:    config.TopicName,
		oauth:        &oauth{Credentials: creds},
	}, nil
}

// Initializes the pubsub client by authenticating and.
func (c *PubSubClient) Initialize(ctx context.Context, config Config) error {
	if err := c.login(); err != nil {
		return err
	}

	sdk.Logger(ctx).Info().Msg("fetched user info")

	topic, err := c.GetTopic(config.TopicName)
	if err != nil {
		return fmt.Errorf("could not fetch topic: %w", err)
	}
	sdk.Logger(ctx).Info().Msgf("got topic %s", topic.TopicName)

	if !topic.GetCanSubscribe() {
		return fmt.Errorf("this user is not allowed to subscribe to the following topic: %s", topic.TopicName)
	}

	c.tomb, _ = tomb.WithContext(ctx)

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

func (c *PubSubClient) login() error {
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
	c.orgID = userInfoResp.UserID

	return nil
}

// Next returns the next record from the buffer.
func (c *PubSubClient) Next(ctx context.Context) (sdk.Record, error) {
	sdk.Logger(ctx).Debug().Msgf("Next - number of records in buffer %d", len(c.buffer))
	sdk.Logger(ctx).Debug().Msgf("Next - context status %v", ctx.Err())

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
		sdk.Logger(ctx).Debug().Msgf("next record - %v", r)
		return r, nil
	}
}

// Stop ends CDC processing.
func (c *PubSubClient) Stop() {
	sdk.Logger(context.Background()).Debug().Msg("Stop - stopping ticker!")
	c.ticker.Stop()
	_ = c.tomb.Killf("cdc iterator is stopping")
}

func (c *PubSubClient) Wait(ctx context.Context) error {
	tctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

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
	sdk.Logger(ctx).Debug().Msg("StartCDC - Starting")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.ticker.C: // detect changes every polling period.
			sdk.Logger(ctx).Debug().Msg("StartCDC - Begin Receiving Events")
			events, err := c.Recv(ctx)

			// if replay ID is invalid, unset it, and try again
			if err != nil && strings.Contains(strings.ToLower(err.Error()), "replay id validation failed") {
				sdk.Logger(ctx).Debug().Msgf("invalid replayID detected: %s", string(c.currReplayID))
				sdk.Logger(ctx).Debug().Msg("attempting Recv() again")
				c.currReplayID = nil
				events, err = c.Recv(ctx)
			}
			if err != nil {
				if !c.authErr(err) {
					return fmt.Errorf("error recv events: %w", err)
				}

				if err := c.login(); err != nil {
					return fmt.Errorf("failed to refresh auth: %w", err)
				}

				break // retry with fresh credentials
			}

			for _, entry := range events {
				sdk.Logger(ctx).Debug().Msg("StartCDC - Build Record")
				output, err := c.buildRecord(entry)
				if err != nil {
					return fmt.Errorf("could not build record for %q: %w", entry.replayID, err)
				}
				c.buffer <- output
			}
		}
	}
}

func (*PubSubClient) authErr(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "upstream connect error") ||
		strings.Contains(msg, "invalid_grant") ||
		strings.Contains(msg, "disconnect") ||
		strings.Contains(msg, "service is unavailable")
}

func (c *PubSubClient) buildRecord(event ConnectResponseEvent) (sdk.Record, error) {
	// TODO - ADD something here to distinguish creates, deletes, updates.
	rec := sdk.SourceUtil{}.NewRecordCreate(
		sdk.Position(event.replayID),
		sdk.Metadata{},
		sdk.StructuredData{
			"replayId": event.replayID,
			"id":       event.eventID,
		},
		sdk.StructuredData(event.Data),
	)

	return rec, nil
}

// Closes the underlying connection to the gRPC server.
func (c *PubSubClient) Close() error {
	return c.conn.Close()
}

// Wrapper function around the GetTopic RPC. This will add the OAuth credentials and make a call to fetch data about a specific topic.
func (c *PubSubClient) GetTopic(topicName string) (*proto.TopicInfo, error) {
	var trailer metadata.MD

	req := &proto.TopicRequest{
		TopicName: topicName,
	}

	ctx, cancelFn := context.WithTimeout(c.getAuthContext(), GRPCCallTimeout)
	defer cancelFn()

	resp, err := c.pubSubClient.GetTopic(ctx, req, grpc.Trailer(&trailer))
	if err != nil {
		return nil, fmt.Errorf("error getting topic from salesforce api - %s", err)
	}

	return resp, nil
}

// Wrapper function around the GetSchema RPC. This will add the OAuth credentials and make a call to fetch data about a specific schema.
func (c *PubSubClient) GetSchema(schemaID string) (*proto.SchemaInfo, error) {
	var trailer metadata.MD

	req := &proto.SchemaRequest{
		SchemaId: schemaID,
	}

	ctx, cancelFn := context.WithTimeout(c.getAuthContext(), GRPCCallTimeout)
	defer cancelFn()

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
	topicName string,
	replayPreset proto.ReplayPreset,
	replayID []byte,
) (proto.PubSub_SubscribeClient, []byte, error) {
	sdk.Logger(ctx).Debug().Msg("Subscribing to pubsub.")
	sdk.Logger(ctx).Debug().Msgf("replayID: %s", string(replayID))
	sdk.Logger(ctx).Debug().Msgf("replayPreset: %s", proto.ReplayPreset_name[int32(replayPreset)])
	sdk.Logger(ctx).Debug().Msgf("topicName: %s", topicName)
	subscribeClient, err := c.pubSubClient.Subscribe(c.getAuthContext())
	if err != nil {
		return nil, replayID, fmt.Errorf("error subscribing to pubsub, %s", err)
	}

	initialFetchRequest := &proto.FetchRequest{
		TopicName:    topicName,
		ReplayPreset: replayPreset,
		NumRequested: 1,
	}

	if replayPreset == proto.ReplayPreset_CUSTOM && replayID != nil {
		initialFetchRequest.ReplayId = replayID
	}

	err = subscribeClient.Send(initialFetchRequest)
	sdk.Logger(ctx).Debug().Msgf("Subcribe pubsub status, %s", err)

	if err == io.EOF {
		sdk.Logger(ctx).Warn().Msg("EOF error returned from initial Send call, proceeding anyway")
		return nil, replayID, fmt.Errorf("EOF error on subscribe send - %s", err)
	} else if err != nil {
		return nil, replayID, fmt.Errorf("error initial fetch request - %s", err)
	}

	return subscribeClient, replayID, nil
}

func (c *PubSubClient) Recv(ctx context.Context) ([]ConnectResponseEvent, error) {
	var err error
	if len(c.currReplayID) > 0 {
		c.subClient, c.currReplayID, err = c.Subscribe(
			ctx,
			c.topicName,
			proto.ReplayPreset_CUSTOM,
			c.currReplayID)
		if err != nil {
			return nil, fmt.Errorf("error subscribing to topic on custom replay id %q: %w", c.currReplayID, err)
		}
	} else {
		c.subClient, c.currReplayID, err = c.Subscribe(
			ctx,
			c.topicName,
			proto.ReplayPreset_LATEST,
			nil)
		if err != nil {
			return nil, fmt.Errorf("error subscribing to topic on latest: %w", err)
		}
	}

	sdk.Logger(ctx).Debug().Msgf("Receive Funk 1 - Waiting for events with replayID %s!", c.currReplayID)

	resp, err := c.subClient.Recv()

	sdk.Logger(ctx).Debug().Msg("Receive Funk 1 - Got events!")
	sdk.Logger(ctx).Debug().Msgf("Received error for replay id %s - %s", c.currReplayID, err)

	if err == io.EOF {
		return nil, fmt.Errorf("stream closed")
	} else if err != nil {
		return nil, fmt.Errorf("error receving events from pubsub client - %s", err)
	}

	var requestedEvents []ConnectResponseEvent
	sdk.Logger(ctx).Debug().Msg(fmt.Sprintf("Receive Funk 4 - Number of events %d", len(resp.Events)))

	for _, event := range resp.Events {
		var fetchedEvent ConnectResponseEvent
		getEvent := event.GetEvent()
		codec, err := c.fetchCodec(ctx, getEvent.SchemaId)
		sdk.Logger(ctx).Debug().Msgf("Receive Funk - fetching codec, err - %s", err)

		if err != nil {
			return requestedEvents, err
		}

		parsed, _, err := codec.NativeFromBinary(getEvent.Payload)
		sdk.Logger(ctx).Debug().Msgf("Receive Funk - parsing payload, err - %s", err)

		if err != nil {
			return requestedEvents, err
		}

		payload, ok := parsed.(map[string]interface{})
		sdk.Logger(ctx).Debug().Msgf("Receive Funk - payload status, err - %s", err)
		if !ok {
			return requestedEvents, fmt.Errorf("receive  - error casting parsed event: %v", payload)
		}

		body := flattenUnionFields(ctx, payload, c.unionFields[getEvent.SchemaId])
		rID := event.GetReplayId()
		fetchedEvent.replayID = rID
		fetchedEvent.eventID = event.Event.Id
		fetchedEvent.Data = body
		requestedEvents = append(requestedEvents, fetchedEvent)
	}

	sdk.Logger(ctx).Debug().Msgf("Receive Funk 8 - Finished Receive, number of events - %d", len(requestedEvents))

	return requestedEvents, nil
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
