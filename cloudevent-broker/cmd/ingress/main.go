/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"knative.dev/pkg/signals"

	cgpubsub "github.com/chainguard-dev/terraform-cloudrun-glue/pkg/pubsub"
)

const (
	retryDelay = 10 * time.Millisecond
	maxRetry   = 3
)

type envConfig struct {
	Port    int    `envconfig:"PORT" default:"8080" required:"true"`
	Project string `envconfig:"PROJECT_ID" required:"true"`
	Topic   string `envconfig:"PUBSUB_TOPIC" required:"true"`
}

func main() {
	ctx := signals.NewContext()

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("failed to process env var: %s", err)
	}

	c, err := cloudevents.NewClientHTTP(cloudevents.WithPort(env.Port))
	if err != nil {
		log.Fatalf("failed to create CE client, %v", err)
	}

	psc, err := pubsub.NewClient(ctx, env.Project, option.WithTokenSource(google.ComputeTokenSource("")))
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
