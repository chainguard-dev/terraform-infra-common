/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/schemas"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/google/go-github/v60/github"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/api/idtoken"
)

type envConfig struct {
	Port          int    `envconfig:"PORT" default:"8080" required:"true"`
	IngressURI    string `envconfig:"EVENT_INGRESS_URI" required:"true"`
	WebhookSecret string `envconfig:"WEBHOOK_SECRET" required:"true"`
}

var types = map[string]interface{}{
	"dev.chainguard.github.workflow_run": schemas.WorkflowRunEvent{},
	"dev.chainguard.github.pull_request": schemas.PullRequestEvent{},
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		clog.Fatalf("failed to process env var: %s", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	log := clog.FromContext(ctx)

	c, err := idtoken.NewClient(ctx, env.IngressURI)
	if err != nil {
		log.Fatalf("failed to create idtoken client: %v", err) //nolint:gocritic
	}
	ceclient, err := cloudevents.NewClientHTTP(
		cloudevents.WithTarget(env.IngressURI),
		cehttp.WithClient(http.Client{Transport: httpmetrics.WrapTransport(c.Transport)}))
	if err != nil {
		log.Fatalf("failed to create cloudevents client: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := clog.FromContext(ctx)

		defer r.Body.Close()

		// https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
		payload, err := github.ValidatePayload(r, []byte(env.WebhookSecret))
		if err != nil {
			log.Errorf("failed to verify webhook: %v", err)
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "failed to verify webhook: %v", err)
			return
		}

		// https://docs.github.com/en/webhooks/webhook-events-and-payloads#delivery-headers
		t := github.WebHookType(r)
		if t == "" {
			log.Errorf("missing X-GitHub-Event header")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		t = "dev.chainguard.github." + t
		log = log.With("event-type", t)

		var obj interface{}
		// If it's a known type, we decode into that type so only the known fields are populated.
		// Otherwise, we decode into a generic interface{} and forward the full event payload.
		if typ, known := types[t]; known {
			// Make a defensive copy of the type so we don't populate the original.
			cp := reflect.New(reflect.TypeOf(typ)).Interface()
			err = json.Unmarshal(payload, &cp)
			obj = cp
		} else {
			err = json.Unmarshal(payload, &obj)
		}
		if err != nil {
			log.Errorf("failed to decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Debugf("forwarding event: %s", t)

		event := cloudevents.NewEvent()
		event.SetType(t)
		event.SetSource(r.Host)
		// TODO: Extract organization and repo to set in subject, for better filtering.
		// event.SetSubject(fmt.Sprintf("%s/%s", org, repo))
		if err := event.SetData(cloudevents.ApplicationJSON, schemas.Wrapper[interface{}]{
			When: time.Now(),
			Body: obj,
		}); err != nil {
			log.Errorf("failed to set data: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		const retryDelay = 10 * time.Millisecond
		const maxRetry = 3
		rctx := cloudevents.ContextWithRetriesExponentialBackoff(context.WithoutCancel(ctx), retryDelay, maxRetry)
		if ceresult := ceclient.Send(rctx, event); cloudevents.IsUndelivered(ceresult) || cloudevents.IsNACK(ceresult) {
			log.Errorf("Failed to deliver event: %v", ceresult)
			w.WriteHeader(http.StatusInternalServerError)
		}
		log.Debugf("event forwarded")
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", env.Port),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatalf("ListenAndServe: %v", srv.ListenAndServe())
}
