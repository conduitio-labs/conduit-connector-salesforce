package source

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
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

var (
	// topic and subscription-related variables

	// gRPC server variables
	GRPCEndpoint    = "api.pubsub.salesforce.com:7443"
	GRPCDialTimeout = 5 * time.Second
	GRPCCallTimeout = 5 * time.Second
)

type PubSubClient struct {
	accessToken string
	instanceURL string

	userID string
	orgID  string

	conn         *grpc.ClientConn
	pubSubClient proto.PubSubClient
	subClient    proto.PubSub_SubscribeClient

	schemaCache map[string]*goavro.Codec

	buffer chan sdk.Record
	caches chan []ConnectResponseEvent
	ticker *time.Ticker

	currReplayId []byte
	tomb         *tomb.Tomb
	topic        string
}

type ConnectResponseEvent struct {
	Data     map[string]interface{}
	ReplayID []byte
}

type Position struct {
	ReplayID  []byte
	TopicName string
}

// Creates a new connection to the gRPC server and returns the wrapper struct
func NewGRPCClient(pollingPeriod time.Duration, config Config, sdkPos sdk.Position) (*PubSubClient, error) {
	fmt.Println("NewGRPCClient - Starting GRPC Client")

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

	ctx, cancelFn := context.WithTimeout(context.Background(), GRPCDialTimeout)
	defer cancelFn()

	conn, err := grpc.DialContext(ctx, GRPCEndpoint, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("error on dialing grpc - %s", err)
	}

	fmt.Println("NewGRPCClient - Initialize Pubsub")

	pubSub := &PubSubClient{
		conn:         conn,
		pubSubClient: proto.NewPubSubClient(conn),
		schemaCache:  make(map[string]*goavro.Codec),
		buffer:       make(chan sdk.Record, 1),
		caches:       make(chan []ConnectResponseEvent),
		ticker:       time.NewTicker(pollingPeriod),
		tomb:         &tomb.Tomb{},
		topic:        config.TopicName,
	}

	creds := Credentials{
		ClientID:      config.ClientID,
		ClientSecret:  config.ClientSecret,
		OAuthEndpoint: config.OAuthEndpoint,
	}

	if err := pubSub.Authenticate(creds); err != nil {
		return nil, fmt.Errorf("could not authenticate: %w", err)
	}

	err = pubSub.FetchUserInfo(config.OAuthEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not fetch user info: %w", err)
	}
	sdk.Logger(ctx).Info().Msg("fetched user info")

	topic, err := pubSub.GetTopic(config.TopicName)
	if err != nil {
		return nil, fmt.Errorf("could not fetch topic: %w", err)
	}
	sdk.Logger(ctx).Info().Msgf("got topic %s", topic.TopicName)

	if !topic.GetCanSubscribe() {
		return nil, fmt.Errorf("this user is not allowed to subscribe to the following topic: %s", topic.TopicName)
	}

	pubSub.tomb.Go(pubSub.startCDC)
	pubSub.tomb.Go(pubSub.flush)

	fmt.Println("NewGRPCClient - Finished starting GRPC Client")
	return pubSub, nil
}

func (c *PubSubClient) HasNext(_ context.Context) bool {
	return len(c.buffer) > 0 || !c.tomb.Alive() // if tomb is dead we return true so caller will fetch error with Next
}

// Next returns the next record from the buffer.
func (c *PubSubClient) Next(ctx context.Context) (sdk.Record, error) {
	select {
	case r := <-c.buffer:
		return r, nil
	case <-c.tomb.Dead():
		return sdk.Record{}, c.tomb.Err()
	case <-ctx.Done():
		return sdk.Record{}, ctx.Err()
	}
}

func (c *PubSubClient) Stop() {
	c.ticker.Stop()
	c.tomb.Kill(errors.New("cdc iterator is stopped"))
}

func (c *PubSubClient) startCDC() error {
	defer close(c.caches)
	fmt.Println("StartCDC - Starting")
	for {
		select {
		case <-c.tomb.Dying():
			return c.tomb.Err()
		case <-c.ticker.C: // detect changes every polling period
			fmt.Println("StartCDC - Begin Receiving Events")
			events, err := c.Recv(c.tomb.Context(nil)) //nolint:staticcheck // SA1012 tomb expects nil
			if err != nil {
				return fmt.Errorf("error receiving events - %s", err)
			}

			select {

			case c.caches <- events:
				if len(events) != 0 {
					c.currReplayId = events[len(events)-1].ReplayID
				} else {
					c.currReplayId = nil
				}

			case <-c.tomb.Dying():
				return c.tomb.Err()
			}
		}
	}
}

func (w *PubSubClient) flush() error {
	defer close(w.buffer)
	for {
		select {
		case <-w.tomb.Dying():
			return w.tomb.Err()
		case cache := <-w.caches:
			for _, entry := range cache {
				fmt.Println("Flush - Build Record")
				output, err := w.buildRecord(entry)
				if err != nil {
					return fmt.Errorf("could not build record for %q: %w", entry.ReplayID, err)
				}
				select {
				case w.buffer <- output:
					// worked fine
				case <-w.tomb.Dying():
					return w.tomb.Err()
				}
			}
		}
	}
}

func (c *PubSubClient) buildRecord(event ConnectResponseEvent) (sdk.Record, error) {
	// TODO - ADD something here to distinguish creates, deletes, updates
	rec := sdk.SourceUtil{}.NewRecordCreate(
		sdk.Position(event.ReplayID),
		sdk.Metadata{},
		sdk.RawData(event.ReplayID),
		sdk.StructuredData(event.Data),
	)

	return rec, nil
}

// Closes the underlying connection to the gRPC server
func (c *PubSubClient) Close() {
	c.conn.Close()
}

// Makes a call to the OAuth server to fetch credentials. Credentials are stored as part of the PubSubClient object so that they can be
// referenced later in other methods
func (c *PubSubClient) Authenticate(creds Credentials) error {
	resp, err := Login(creds)
	if err != nil {
		return err
	}

	c.accessToken = resp.AccessToken
	c.instanceURL = resp.InstanceURL

	return nil
}

// Makes a call to the OAuth server to fetch user info. User info is stored as part of the PubSubClient object so that it can be referenced
// later in other methods
func (c *PubSubClient) FetchUserInfo(oauthEndpoint string) error {
	resp, err := UserInfo(oauthEndpoint, c.accessToken)
	if err != nil {
		return fmt.Errorf("error getting user info from oauth server - %s", err)
	}

	c.userID = resp.UserID
	c.orgID = resp.OrganizationID

	return nil
}

// Wrapper function around the GetTopic RPC. This will add the OAuth credentials and make a call to fetch data about a specific topic
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

// Wrapper function around the GetSchema RPC. This will add the OAuth credentials and make a call to fetch data about a specific schema
func (c *PubSubClient) GetSchema(schemaId string) (*proto.SchemaInfo, error) {
	var trailer metadata.MD

	req := &proto.SchemaRequest{
		SchemaId: schemaId,
	}

	ctx, cancelFn := context.WithTimeout(c.getAuthContext(), GRPCCallTimeout)
	defer cancelFn()

	resp, err := c.pubSubClient.GetSchema(ctx, req, grpc.Trailer(&trailer))
	if err != nil {
		return nil, fmt.Errorf("error getting schema from salesforce api - %s", err)
	}

	return resp, nil
}

// Wrapper function around the Subscribe RPC. This will add the OAuth credentials and create a separate streaming client that will be used to
// fetch data from the topic. This method will continuously consume messages unless an error occurs; if an error does occur then this method will
// return the last successfully consumed ReplayId as well as the error message. If no messages were successfully consumed then this method will return
// the same ReplayId that it originally received as a parameter
func (c *PubSubClient) Subscribe(
	ctx context.Context,
	topicName string,
	replayPreset proto.ReplayPreset,
	replayId []byte,
) (proto.PubSub_SubscribeClient, []byte, error) {
	subscribeClient, err := c.pubSubClient.Subscribe(c.getAuthContext())
	if err != nil {
		return nil, replayId, err
	}

	initialFetchRequest := &proto.FetchRequest{
		TopicName:    topicName,
		ReplayPreset: replayPreset,
		NumRequested: 1,
	}

	if replayPreset == proto.ReplayPreset_CUSTOM && replayId != nil {
		initialFetchRequest.ReplayId = replayId
	}

	err = subscribeClient.Send(initialFetchRequest)
	if err == io.EOF {
		sdk.Logger(ctx).Warn().Msg("EOF error returned from initial Send call, proceeding anyway")
		return nil, replayId, fmt.Errorf("EOF error on subscribe send - %s", err)
	} else if err != nil {
		return nil, replayId, fmt.Errorf("error initial fetch request - %s", err)
	}

	return subscribeClient, replayId, nil
}

func (c *PubSubClient) Recv(
	ctx context.Context,
) ([]ConnectResponseEvent, error) {
	var err error
	if len(c.currReplayId) > 0 {
		c.subClient, c.currReplayId, err = c.Subscribe(
			ctx,
			c.topic,
			proto.ReplayPreset_CUSTOM,
			c.currReplayId)
		if err != nil {
			fmt.Println(err)
			return nil, fmt.Errorf("error subscribing to topic on custom replay id %s - %s", c.currReplayId, err)
		}
	} else {
		c.subClient, c.currReplayId, err = c.Subscribe(
			ctx,
			c.topic,
			proto.ReplayPreset_LATEST,
			nil)
		if err != nil {
			fmt.Println(err)
			return nil, fmt.Errorf("error subscribing to topic on latest - %s", err)
		}
	}
	fmt.Println("Receive Funk 1 - Waiting for events!")

	resp, err := c.subClient.Recv()

	if err == io.EOF {
		return nil, fmt.Errorf("stream closed")
	} else if err != nil {
		return nil, fmt.Errorf("error receving events from pubsub client - %s", err)
	}

	var requestedEvents []ConnectResponseEvent
	fmt.Println("Receive Funk 4 - Receving events!")

	for _, event := range resp.Events {
		var fetchedEvent ConnectResponseEvent
		getEvent := event.GetEvent()
		codec, err := c.fetchCodec(ctx, getEvent.SchemaId)
		if err != nil {
			return requestedEvents, err
		}

		parsed, _, err := codec.NativeFromBinary(getEvent.Payload)
		if err != nil {
			return requestedEvents, err
		}

		body, ok := parsed.(map[string]interface{})
		if !ok {
			return requestedEvents, fmt.Errorf("receive  - error casting parsed event: %v", body)
		}
		rID := event.GetReplayId()
		fetchedEvent.ReplayID = rID
		fetchedEvent.Data = body
		requestedEvents = append(requestedEvents, fetchedEvent)
	}

	fmt.Println("Receive Funk 8 - Finished Receive")

	return requestedEvents, nil
}

// Unexported helper function to retrieve the cached codec from the PubSubClient's schema cache. If the schema ID is not found in the cache
// then a GetSchema call is made and the corresponding codec is cached for future use
func (c *PubSubClient) fetchCodec(ctx context.Context, schemaId string) (*goavro.Codec, error) {
	codec, ok := c.schemaCache[schemaId]
	if ok {
		sdk.Logger(ctx).Trace().Msg("Fetched cached codec...")
		return codec, nil
	}

	sdk.Logger(ctx).Trace().Msg("Making GetSchema request for uncached schema...")
	schema, err := c.GetSchema(schemaId)
	if err != nil {
		return nil, fmt.Errorf("error making getschema request for uncached schema - %s", err)
	}

	sdk.Logger(ctx).Trace().Msg("Creating codec from uncached schema...")
	codec, err = goavro.NewCodec(schema.GetSchemaJson())
	if err != nil {
		return nil, fmt.Errorf("error creating codec from uncached schema - %s", err)
	}

	c.schemaCache[schemaId] = codec

	return codec, nil
}

const (
	tokenHeader    = "accesstoken"
	instanceHeader = "instanceurl"
	tenantHeader   = "tenantid"
)

// Returns a new context with the necessary authentication parameters for the gRPC server
func (c *PubSubClient) getAuthContext() context.Context {
	pairs := metadata.Pairs(
		tokenHeader, c.accessToken,
		instanceHeader, c.instanceURL,
		tenantHeader, c.orgID,
	)

	return metadata.NewOutgoingContext(context.Background(), pairs)
}

// Fetches system certs and returns them if possible. If unable to fetch system certs then an empty cert pool is returned instead
func getCerts() *x509.CertPool {
	if certs, err := x509.SystemCertPool(); err == nil {
		return certs
	}

	return x509.NewCertPool()
}
