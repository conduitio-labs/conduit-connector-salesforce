package common

import (
	"os"
	"time"

	"github.com/conduitio-labs/conduit-connector-salesforce/pubsub/proto"
)

var (
	// topic and subscription-related variables
	TopicName           = "/event/CarMaintenance__e"
	ReplayPreset        = proto.ReplayPreset_EARLIEST
	ReplayId     []byte = nil
	Appetite     int32  = 5

	// gRPC server variables
	GRPCEndpoint    = "api.pubsub.salesforce.com:7443"
	GRPCDialTimeout = 5 * time.Second
	GRPCCallTimeout = 5 * time.Second

	// OAuth header variables
	GrantType    = "password"
	ClientId     = os.Getenv("SF_CLIENT_ID")
	ClientSecret = os.Getenv("SF_CLIENT_SECRET")
	Username     = os.Getenv("SF_USERNAME")

	// OAuth server variables
	OAuthEndpoint    = os.Getenv("SF_OAUTH_ENDPOINT")
	OAuthDialTimeout = 5 * time.Second
)
