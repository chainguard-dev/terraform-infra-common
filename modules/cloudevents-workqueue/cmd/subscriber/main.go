/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Port             int    `env:"PORT,default=8080"`
	WorkqueueService string `env:"WORKQUEUE_SERVICE,required"`
	ExtensionKey     string `env:"EXTENSION_KEY,required"`
}

func main() {
	ctx := context.Background()
	logger := clog.FromContext(ctx)

	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		logger.Fatalf("Failed to process configuration: %v", err)
	}

	logger.With(
		"port", cfg.Port,
		"workqueue_service", cfg.WorkqueueService,
		"extension_key", cfg.ExtensionKey,
	).Info("Starting CloudEvents to Workqueue subscriber")

	// Create workqueue client
	queueClient, err := workqueue.NewWorkqueueClient(ctx, cfg.WorkqueueService)
	if err != nil {
		logger.Fatalf("Failed to create workqueue client: %v", err)
	}
	defer queueClient.Close()

	// Create CloudEvents client
	p, err := cloudevents.NewHTTP(
		cloudevents.WithPort(cfg.Port),
		cloudevents.WithPath("/"),
	)
	if err != nil {
		logger.Fatalf("Failed to create CloudEvents HTTP transport: %v", err)
	}

	c, err := cloudevents.NewClient(p)
	if err != nil {
		logger.Fatalf("Failed to create CloudEvents client: %v", err)
	}

	// Create handler
	handler := &eventHandler{
		queueClient:  queueClient,
		extensionKey: cfg.ExtensionKey,
	}

	// Start receiving events
	logger.Info("Ready to receive CloudEvents")
	if err := c.StartReceiver(ctx, handler.handleEvent); err != nil {
		logger.Fatalf("Failed to start CloudEvents receiver: %v", err)
	}
}

type eventHandler struct {
	queueClient  workqueue.Client
	extensionKey string
}

func (h *eventHandler) handleEvent(ctx context.Context, event cloudevents.Event) error {
	logger := clog.FromContext(ctx).With(
		"event_id", event.ID(),
		"event_type", event.Type(),
		"event_source", event.Source(),
		"event_subject", event.Subject(),
	)

	logger.Debug("Received CloudEvent")

	// Extract the workqueue key from the specified extension
	extensions := event.Extensions()
	keyValue, ok := extensions[h.extensionKey]
	if !ok {
		logger.With("extension_key", h.extensionKey).Warn("Extension key not found in event, skipping")
		// Return success to acknowledge the event (we don't want to retry)
		return nil
	}

	key, ok := keyValue.(string)
	if !ok {
		logger.With(
			"extension_key", h.extensionKey,
			"extension_value", keyValue,
			"extension_type", fmt.Sprintf("%T", keyValue),
		).Error("Extension value is not a string")
		// Return success to acknowledge the event (we don't want to retry)
		return nil
	}

	if key == "" {
		logger.With("extension_key", h.extensionKey).Warn("Extension value is empty, skipping")
		// Return success to acknowledge the event (we don't want to retry)
		return nil
	}

	logger = logger.With("workqueue_key", key)

	// Queue the work item
	_, err := h.queueClient.Process(ctx, &workqueue.ProcessRequest{
		Key:      key,
		Priority: 0, // Default priority
	})
	if err != nil {
		logger.Errorf("Failed to queue work item: %v", err)
		// Return error to trigger pubsub retry
		return fmt.Errorf("failed to queue work item: %w", err)
	}

	logger.Info("Successfully queued work item")
	return nil
}

// Health check endpoint (CloudEvents client will set this up at /health/ready)
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK\n")); err != nil {
		log.Printf("Failed to write health response: %v", err)
	}
}

func init() {
	// Add a health check endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.Method == http.MethodGet {
			healthHandler(w, r)
			return
		}
		// Let CloudEvents handle other requests
		http.NotFound(w, r)
	})
}
