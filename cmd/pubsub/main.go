package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	source "github.com/conduitio-labs/conduit-connector-salesforce/source_pubsub"
)

func init() {
	log := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.DefaultContextLogger = &log
}

func main() {
	ctx := context.Background()
	s := source.New()

	cfg := map[string]string{
		"clientID":      os.Getenv("SF_CLIENT_ID"),
		"clientSecret":  os.Getenv("SF_CLIENT_SECRET"),
		"username":      os.Getenv("SF_USERNAME"),
		"oauthEndpoint": os.Getenv("SF_OAUTH_ENDPOINT"),
		"topicName":     "/event/CarMaintenance__e",
	}

	err := s.Configure(ctx, cfg)
	if err != nil {
		panic(err)
	}

	err = s.Open(ctx, nil)
	if err != nil {
		panic(err)
	}

	rec, err := s.Read(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(rec.Payload.After.Bytes()))

	err = s.Ack(ctx, nil)
	if err != nil {
		panic(err)
	}
}
