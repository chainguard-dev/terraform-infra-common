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
	"syscall"
	"time"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"google.golang.org/api/storage/v1"

	"github.com/kelseyhightower/envconfig"
)

type envConfig struct {
	Port       int    `envconfig:"PORT" default:"8080" required:"true"`
	IngressURI string `envconfig:"INGRESS_URI" required:"true"`
}

var eventTypes = map[string]string{
	"OBJECT_FINALIZE":        "dev.chainguard.storage.object.finalize",
	"OBJECT_METADATA_UPDATE": "dev.chainguard.storage.object.metadata_update",
	"OBJECT_DELETE":          "dev.chainguard.storage.object.delete",
	"OBJECT_ARCHIVE":         "dev.chainguard.storage.object.archive",
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		clog.Fatalf("failed to process env var: %s", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	clog.DebugContextf(ctx, "env: %+v", env)

	ceclient, err := mce.NewClientHTTP("trampoline", mce.WithTarget(ctx, env.IngressURI)...)
	if err != nil {
		clog.FatalContextf(ctx, "failed to create cloudevents client: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := clog.FromContext(ctx)

		defer r.Body.Close()
		var obj storage.Object
		if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
			log.Errorf("failed to decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		t, ok := eventTypes[r.Header.Get("Eventtype")]
		if !ok {
			log.Errorf("unknown event type: %s", r.Header.Get("Eventtype"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bucket := r.Header.Get("Bucketid")
		object := r.Header.Get("Objectid")
		log = log.With(
			"event-type", t,
			"bucket", bucket,
			"object", object,
		)
		log.Debugf("forwarding event: %s", r.Header.Get("Eventtype"))

		event := cloudevents.NewEvent()
		event.SetType(t)
		event.SetSubject(fmt.Sprintf("%s/%s", bucket, object))
		event.SetSource(r.Host)
		event.SetExtension("bucket", r.Header.Get("Bucketid"))
		event.SetExtension("object", r.Header.Get("Objectid"))
		if err := event.SetData(cloudevents.ApplicationJSON, obj); err != nil {
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
	clog.FatalContextf(ctx, "ListenAndServe: %v", srv.ListenAndServe())
}
