# What's this?

What's here is an implementation of the new salesforce source connector. The source code has been taken from [here](https://github.com/forcedotcom/pub-sub-api/tree/main/go) and modified to adapt to the conduit connector workflow and to be secure. The original source code used username:password auth to authenticate, but the current connector uses [client credentials flow](https://help.salesforce.com/s/articleView?id=sf.remoteaccess_oauth_client_credentials_flow.htm&type=5).

# How to publish an event.

- Run `git clone https://github.com/forcedotcom/pub-sub-api`
- cd into the cloned directory.
- Follow the prerequisites from [here](https://github.com/forcedotcom/pub-sub-api/tree/main/go#prerequisites)
- Follow the execution from [here](https://github.com/forcedotcom/pub-sub-api/tree/main/go#execution)

- there's an example, `go/examples/publish`. Run this to test publishing a custom app platform event.

# How to run the source connector

- After having followed the prerequisites, you should be able to add the
  missing parameters to this code example. After filling them up, run this main
  function. This code example is also on `cmd/pubsub/main.go`

```go
package main

import (
	"context"
	"fmt"

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
		"clientID":      "SF_CLIENT_ID",
		"clientSecret":  "SF_CLIENT_SECRET",
		"username":      "SF_USERNAME",
		"oauthEndpoint": "SF_OAUTH_ENDPOINT",
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
```
