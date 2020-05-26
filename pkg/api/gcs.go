package api

import (
	"context"
	"log"

	"cloud.google.com/go/storage"
)

var gcsClient *storage.Client

func CreateGCSClient() *storage.Client {
	ctx := context.Background()

	// Creates a client.
	var err error
	gcsClient, err = storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return gcsClient
}
