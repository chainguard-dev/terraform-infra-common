/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/pubsub/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sethvargo/go-envconfig"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"chainguard.dev/go-grpc-kit/pkg/options"
	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	"github.com/chainguard-dev/terraform-infra-common/pkg/profiler"
	cgpubsub "github.com/chainguard-dev/terraform-infra-common/pkg/pubsub"
)

const (
	retryDelay = 10 * time.Millisecond
	maxRetry   = 3
)

// buildPublishers creates one publisher per distinct destination topic — the
// default topic plus each override target — reusing a single publisher when
// types share a topic or an override points back at the default. It returns the
// default publisher, a per-type routing map (types absent from it fall back to
// the default), and the full set of distinct publishers to Stop() on shutdown.
// Every publisher enables message ordering, which is required to publish any
// message carrying an OrderingKey; keyless messages are unaffected. It errors on
// an empty override topic name rather than deferring the failure to publish time.
func buildPublishers(client *pubsub.Client, defaultTopic string, overrides map[string]string) (def *pubsub.Publisher, byType map[string]*pubsub.Publisher, all []*pubsub.Publisher, err error) {
	byTopic := make(map[string]*pubsub.Publisher, len(overrides)+1)
	publisher := func(name string) *pubsub.Publisher {
		if p, ok := byTopic[name]; ok {
			return p
		}
		p := client.Publisher(name)
		p.EnableMessageOrdering = true
		byTopic[name] = p
		return p
	}
	def = publisher(defaultTopic)
	byType = make(map[string]*pubsub.Publisher, len(overrides))
	for t, name := range overrides {
		if name == "" {
			return nil, nil, nil, fmt.Errorf("override for type %q has an empty topic name", t)
		}
		byType[t] = publisher(name)
	}
	all = make([]*pubsub.Publisher, 0, len(byTopic))
	for _, p := range byTopic {
		all = append(all, p)
	}
	return def, byType, all, nil
}

// publisherForType returns the publisher an event of the given type publishes
// to: its dedicated override publisher when one is configured, otherwise the
// default.
func publisherForType(byType map[string]*pubsub.Publisher, def *pubsub.Publisher, eventType string) *pubsub.Publisher {
	if p, ok := byType[eventType]; ok {
		return p
	}
	return def
}

func main() {
	profiler.SetupProfiler()

	env := envconfig.MustProcess(context.Background(), &struct {
		Port  int    `env:"PORT, default=8080"`
		Topic string `env:"PUBSUB_TOPIC, required"`
		// EventTypeTopicOverrides maps a CloudEvent type to the dedicated topic
		// its events are published to instead of the default Topic, as
		// "type:topic,type:topic". Absent (the default) sends every event to
		// Topic.
		EventTypeTopicOverrides map[string]string `env:"EVENT_TYPE_TOPIC_OVERRIDES"`
	}{})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	c, err := mce.NewClientHTTP("ce-ingress", cloudevents.WithPort(env.Port),
		cehttp.WithRequestDataAtContextMiddleware() /* give request headers to the handler context */)
	if err != nil {
		clog.Fatalf("failed to create CE client, %v", err)
	}

	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		clog.Fatalf("failed to create OIDC provider, %v", err)
	}
	verifier := provider.Verifier(&oidc.Config{
		// When on Cloud Run, this is checked by the platform.
		SkipClientIDCheck: true,
	})

	projectID, err := metadata.ProjectIDWithContext(ctx)
	if err != nil {
		clog.Fatalf("failed to get project ID, %v", err)
	}
	psc, err := pubsub.NewClientWithConfig(ctx, projectID,
		&pubsub.ClientConfig{
			EnableOpenTelemetryTracing: true,
		},
		append(options.ClientOptions(), option.WithTokenSource(google.ComputeTokenSource("")))...)
	if err != nil {
		clog.Fatalf("failed to create pubsub client, %v", err)
	}

	defaultTopic, publisherByType, allPublishers, err := buildPublishers(psc, env.Topic, env.EventTypeTopicOverrides)
	if err != nil {
		clog.Fatalf("failed to build publishers, %v", err)
	}
	defer func() {
		for _, p := range allPublishers {
			p.Stop()
		}
	}()

	clog.InfoContextf(ctx, "broker ingress routing: default topic %q, type overrides %v", env.Topic, env.EventTypeTopicOverrides)

	if err := c.StartReceiver(cloudevents.ContextWithRetriesExponentialBackoff(ctx, retryDelay, maxRetry), func(ctx context.Context, event cloudevents.Event) error {
		// We expect Chainguard webhooks to pass an Authorization header.
		auth := strings.TrimPrefix(cehttp.RequestDataFromContext(ctx).Header.Get("Authorization"), "Bearer ")
		if auth == "" {
			return cloudevents.NewHTTPResult(http.StatusUnauthorized, "Unauthorized")
		}
		tok, err := verifier.Verify(ctx, auth)
		if err != nil {
			clog.ErrorContextf(ctx, "failed to verify Authorization: %v", err)
			return cloudevents.NewHTTPResult(http.StatusUnauthorized, err.Error())
		}
		var claims struct {
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
		}
		if err := tok.Claims(&claims); err != nil {
			clog.ErrorContextf(ctx, "failed to extract email claims: %v", err)
			return cloudevents.NewHTTPResult(http.StatusUnauthorized, err.Error())
		}
		if !claims.EmailVerified {
			clog.ErrorContextf(ctx, "email claim is not verified: %s", claims.Email)
			return cloudevents.NewHTTPResult(http.StatusUnauthorized, "Unverified email claim")
		}
		// Route the event to its dedicated topic if one is configured for its
		// type, otherwise to the default topic.
		return forward(ctx, publisherForType(publisherByType, defaultTopic, event.Type()), event, claims.Email)
	}); err != nil {
		clog.Fatalf("failed to start receiver, %v", err)
	}
}

// forward publishes a verified CloudEvent to the broker topic, stamping the
// authenticated sender's email as the ce-actor extension. The event's
// partitionkey extension becomes the message ordering key.
func forward(ctx context.Context, topic *pubsub.Publisher, event cloudevents.Event, actor string) error {
	msg := cgpubsub.FromCloudEventWithOrdering(ctx, event)
	msg.Attributes["ce-actor"] = actor

	res := topic.Publish(ctx, msg)
	if _, err := res.Get(ctx); err != nil {
		// A failed publish pauses the ordering key in the client. Resume it
		// so later messages for the key keep publishing. This favours
		// liveness: the source's retry of the failed event can land after
		// newer events for the key, so consumers must tolerate reordering
		// around a publish failure.
		if msg.OrderingKey != "" {
			topic.ResumePublish(msg.OrderingKey)
		}
		clog.ErrorContextf(ctx, "failed to forward event: %v\n%v", event, err)
		return cloudevents.NewHTTPResult(http.StatusInternalServerError, err.Error())
	}

	return nil
}
