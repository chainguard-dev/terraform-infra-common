/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/pubsub"
	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"golang.org/x/oauth2/google"

	"github.com/kelseyhightower/envconfig"
	"google.golang.org/api/option"
)

type envConfig struct {
	Port  int    `envconfig:"PORT" default:"8080" required:"true"`
	Topic string `envconfig:"PUBSUB_TOPIC" required:"true"`
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
		log.Panicf("failed to process env var: %s", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	projectID, err := metadata.ProjectID()
	if err != nil {
		log.Panicf("failed to get project ID, %v", err)
	}
	psc, err := pubsub.NewClient(ctx, projectID, option.WithTokenSource(google.ComputeTokenSource("")))
	if err != nil {
		log.Panicf("failed to create pubsub client, %v", err)
	}

	topic := psc.Topic(env.Topic)
	defer topic.Stop()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := clog.FromContext(ctx)

		defer r.Body.Close()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Errorf("failed to read body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		t, ok := eventTypes[r.Header.Get("Eventtype")]
		if !ok {
			log.Errorf("unknown event type: %s", r.Header.Get("Eventtype"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log = log.With(
			"event-type", t,
			"bucket", r.Header.Get("Bucketid"),
			"object", r.Header.Get("Objectid"))
		log.Infof("forwarding event: %s", r.Header.Get("Eventtype"))

		res := topic.Publish(ctx, toMessage(r.Header, t, data))
		if _, err := res.Get(ctx); err != nil {
			log.Errorf("failed to forward event: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	http.ListenAndServe(fmt.Sprintf(":%d", env.Port), nil)
}

func toMessage(hdrs http.Header, eventType string, data []byte) *pubsub.Message {
	return &pubsub.Message{
		Attributes: map[string]string{
			"ce-bucket":    hdrs.Get("Bucketid"),
			"ce-object":    hdrs.Get("Objectid"),
			"ce-type":      eventType,
			"ce-source":    hdrs.Get("Id"),
			"ce-subject":   hdrs.Get("Subject"),
			"ce-time":      hdrs.Get("Updated"),
			"content-type": hdrs.Get("Content-Type"),
		},
		Data: data,
	}
}
