package main

import (
	"log"

	"github.com/conduitio-labs/conduit-connector-salesforce/pubsub/common"
	"github.com/conduitio-labs/conduit-connector-salesforce/pubsub/grpcclient"
)

func main() {
	log.Printf("Creating gRPC client...")
	client, err := grpcclient.NewGRPCClient()
	if err != nil {
		log.Fatalf("could not create gRPC client: %v", err)
	}
	defer client.Close()

	log.Printf("Populating auth token...")
	err = client.Authenticate()
	if err != nil {
		log.Fatalf("could not authenticate: %v", err)
	}

	log.Printf("Populating user info...")
	err = client.FetchUserInfo()
	if err != nil {
		log.Fatalf("could not fetch user info: %v", err)
	}

	log.Printf("Making GetTopic request...")
	topic, err := client.GetTopic()
	if err != nil {
		log.Fatalf("could not fetch topic: %v", err)
	}

	if !topic.GetCanPublish() {
		log.Fatalf("this user is not allowed to publish to the following topic: %s", common.TopicName)
	}

	log.Printf("Making GetSchema request...")
	schema, err := client.GetSchema(topic.GetSchemaId())
	if err != nil {
		log.Fatalf("could not fetch schema: %v", err)
	}

	err = client.Publish(schema)
	if err != nil {
		log.Fatalf("could not publish event: %v", err)
	}

	log.Printf("successfully published event")
}
