package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/miquido/conduit-connector-salesforce/internal/cometd"
	"github.com/miquido/conduit-connector-salesforce/internal/cometd/responses"
	"github.com/miquido/conduit-connector-salesforce/internal/salesforce/oauth"
)

const sfCometDVersion = "54.0"

var subscriptions = make(map[string]func(event responses.ConnectResponseEvent))

func main() {
	// Auth
	oAuth := oauth.NewClient(
		os.Getenv("ENVIRONMENT"),
		os.Getenv("CLIENT_ID"),
		os.Getenv("CLIENT_SECRET"),
		os.Getenv("USERNAME"),
		os.Getenv("PASSWORD"),
		os.Getenv("SECURITY_TOKEN"),
	)

	token, err := oAuth.Authenticate(context.Background())
	if err != nil {
		panic(err)
	}

	// Streaming API
	streamingClient, err := cometd.NewClient(
		fmt.Sprintf("%s/cometd/%s", token.InstanceURL, sfCometDVersion),
		token.AccessToken,
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", streamingClient)

	// Handshake
	if _, err := streamingClient.Handshake(context.Background()); err != nil {
		panic(err)
	}

	// Subscribe to topic
	subscribeResponse, err := streamingClient.SubscribeToPushTopic(context.Background(), "TaskUpdates")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n%s\n", subscribeResponse, subscribeResponse.GetSubscriptions())
	if !subscribeResponse.Successful {
		panic(subscribeResponse.Error)
	}

	for _, s := range subscribeResponse.GetSubscriptions() {
		subscriptions[s] = func(event responses.ConnectResponseEvent) {
			fmt.Println("PushTopic received", event)
		}
	}

	// Subscription
	timeout := time.After(time.Minute)
	events := make(chan responses.ConnectResponseEvent)

	go func() {
		time.AfterFunc(time.Second*10, func() {
			_, _ = streamingClient.Disconnect(context.Background())
		})

		for {
			if len(subscriptions) == 0 {
				continue
			}

			connectResponse, err := streamingClient.Connect(context.Background())
			if err != nil {
				panic(err)
			}
			if !connectResponse.Successful {
				panic(connectResponse.Error)
			}

			for _, event := range connectResponse.Events {
				events <- event
			}

			if nil != connectResponse.Advice && responses.AdviceReconnectHandshake == connectResponse.Advice.Reconnect {
				fmt.Println("Worker: reconnecting")

				if _, err := streamingClient.Handshake(context.Background()); err != nil {
					panic(err)
				}
			}

			if nil != connectResponse.Advice && responses.AdviceReconnectNone == connectResponse.Advice.Reconnect {
				close(events)

				fmt.Println("Worker: gracefully shutting down")

				break
			}

			if nil != connectResponse.Advice && connectResponse.Advice.Interval > 0 {
				time.Sleep(time.Millisecond * time.Duration(connectResponse.Advice.Interval))
			}
		}
	}()

worker:
	for {
		select {
		case event, ok := <-events:
			if !ok {
				fmt.Println("Worker: event: closed")

				break worker
			}

			fmt.Println("Worker: event", event)

			if callback, exists := subscriptions[event.Channel]; exists {
				callback(event)
			}

		case <-timeout:
			fmt.Println("Worker: timeout")

			break worker
		}
	}

	fmt.Println("Worker has stopped")

	// Unsubscribe
	unsubscribeResponse, err := streamingClient.UnsubscribeToPushTopic(context.Background(), "TaskUpdates")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", unsubscribeResponse)
	if !unsubscribeResponse.Successful {
		panic(unsubscribeResponse.Error)
	}

	// Disconnect
	disconnectResponse, err := streamingClient.Disconnect(context.Background())
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", disconnectResponse)
	if !disconnectResponse.Successful {
		panic(disconnectResponse.Error)
	}
}
