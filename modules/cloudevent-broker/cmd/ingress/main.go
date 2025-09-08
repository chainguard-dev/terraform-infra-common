/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
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

var env = envconfig.MustProcess(context.Background(), &struct {
	Port  int    `env:"PORT, default=8080"`
	Topic string `env:"PUBSUB_TOPIC, required"`
}{})

func main() {
	profiler.SetupProfiler()

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

	topic := psc.Publisher(env.Topic)
	defer topic.Stop()

	if err := c.StartReceiver(cloudevents.ContextWithRetriesExponentialBackoff(ctx, retryDelay, maxRetry), func(ctx context.Context, event cloudevents.Event) error {
		// We expect Chainguard webhooks to pass an Authorization header.
		auth := strings.TrimPrefix(cehttp.RequestDataFromContext(ctx).Header.Get("Authorization"), "Bearer ")
		if auth == "" {
			return cloudevents.NewHTTPResult(http.StatusUnauthorized, "Unauthorized")
		}
		tok, err := verifier.Verify(ctx, auth)
		if err != nil {
			clog.FromContext(ctx).Errorf("failed to verify Authorization: %v", err)
			return cloudevents.NewHTTPResult(http.StatusUnauthorized, err.Error())
		}
		var claims struct {
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
		}
		if err := tok.Claims(&claims); err != nil {
			clog.FromContext(ctx).Errorf("failed to extract email claims: %v", err)
			return cloudevents.NewHTTPResult(http.StatusUnauthorized, err.Error())
		}
		if !claims.EmailVerified {
			clog.FromContext(ctx).Errorf("email claim is not verified: %s", claims.Email)
			return cloudevents.NewHTTPResult(http.StatusUnauthorized, "Unverified email claim")
		}
		msg := cgpubsub.FromCloudEvent(ctx, event)
		// Turn the email of the Google Service Account that sent the cloud
		// event into an "actor" extension on the cloud event.
		msg.Attributes["ce-actor"] = claims.Email

		res := topic.Publish(ctx, msg)
		if _, err := res.Get(ctx); err != nil {
			clog.FromContext(ctx).Errorf("failed to forward event: %v\n%v", event, err)
			return cloudevents.NewHTTPResult(http.StatusInternalServerError, err.Error())
		}
		return nil
	}); err != nil {
		clog.Fatalf("failed to start receiver, %v", err)
	}
}
