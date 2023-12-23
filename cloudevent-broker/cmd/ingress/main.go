/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/pubsub"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	cgpubsub "github.com/chainguard-dev/terraform-cloudrun-glue/pkg/pubsub"
)

const (
	retryDelay = 10 * time.Millisecond
	maxRetry   = 3
)

type envConfig struct {
	Port  int    `envconfig:"PORT" default:"8080" required:"true"`
	Topic string `envconfig:"PUBSUB_TOPIC" required:"true"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("failed to process env var: %s", err)
	}

	c, err := cloudevents.NewClientHTTP(cloudevents.WithPort(env.Port))
	if err != nil {
		log.Fatalf("failed to create CE client, %v", err)
	}

	projectID, err := metadata.ProjectID()
	if err != nil {
		log.Fatalf("failed to get project ID, %v", err)
	}
	psc, err := pubsub.NewClient(ctx, projectID, option.WithTokenSource(google.ComputeTokenSource("")))
	if err != nil {
		log.Fatalf("failed to create pubsub client, %v", err)
	}

	topic := psc.Topic(env.Topic)
	defer topic.Stop()

	if err := c.StartReceiver(cloudevents.ContextWithRetriesExponentialBackoff(ctx, retryDelay, maxRetry), func(ctx context.Context, event cloudevents.Event) {
		res := topic.Publish(ctx, cgpubsub.FromCloudEvent(ctx, event))
		if _, err := res.Get(ctx); err != nil {
			log.Printf("failed to forward event: %v\n%v", err, event)
		}
	}); err != nil {
		log.Panic(err)
	}
}
