/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/chainguard-dev/terraform-infra-common/pkg/profiler"
	"github.com/sethvargo/go-envconfig"
)

// Config holds the application configuration
type Config struct {
	Port               int    `env:"PORT,default=8080"`
	SlackWebhookSecret string `env:"SLACK_WEBHOOK_SECRET,required"`
	SlackChannel       string `env:"SLACK_CHANNEL,required"`
	MessageTemplate    string `env:"MESSAGE_TEMPLATE,required"`
	ProjectID          string `env:"PROJECT_ID,required"`
}

// PubSubMessage represents the structure of a Pub/Sub push message
type PubSubMessage struct {
	Message struct {
		Data        string            `json:"data"`
		Attributes  map[string]string `json:"attributes"`
		MessageID   string            `json:"messageId"`
		PublishTime string            `json:"publishTime"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

// SlackService handles Slack message sending
type SlackService struct {
	webhookURL      string
	channel         string
	messageTemplate string
}

func main() {
	// Set up graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx = clog.WithLogger(ctx, clog.New(slog.Default().Handler()))

	// Process environment configuration
	var env Config
	if err := envconfig.Process(ctx, &env); err != nil {
		clog.FatalContextf(ctx, "failed to process environment configuration: %v", err)
	}

	// Set up profiler
	profiler.SetupProfiler()

	// Start metrics server
	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	// Get Slack webhook URL from Secret Manager
	webhookURL, err := getSecretValue(ctx, env.ProjectID, env.SlackWebhookSecret)
	if err != nil {
		clog.FatalContextf(ctx, "failed to get Slack webhook URL: %v", err)
	}

	// Initialize Slack service
	slackService := &SlackService{
		webhookURL:      webhookURL,
		channel:         env.SlackChannel,
		messageTemplate: env.MessageTemplate,
	}

	// Create HTTP server with standard configuration
	mux := http.NewServeMux()

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	// Add main handler with metrics wrapper
	mux.Handle("/", httpmetrics.Handler("pubsub-slack-bridge", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if err := handlePubSubMessage(ctx, slackService, w, r); err != nil {
			clog.ErrorContext(ctx, "failed to handle Pub/Sub message", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})))

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", env.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	clog.InfoContext(ctx, "Starting server", "port", env.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		clog.FatalContextf(ctx, "server failed: %v", err)
	}
}

// getSecretValue retrieves a secret value from Google Secret Manager
func getSecretValue(ctx context.Context, projectID, secretID string) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID),
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %w", err)
	}

	return string(result.Payload.Data), nil
}

// processMessage processes a message using a Go text template
func processMessage(data map[string]interface{}, templateStr string) (string, error) {
	tmpl, err := template.New("slack").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

// handlePubSubMessage processes incoming Pub/Sub push messages
func handlePubSubMessage(ctx context.Context, slackService *SlackService, w http.ResponseWriter, r *http.Request) error {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Parse the Pub/Sub message
	var pubsubMsg PubSubMessage
	if err := json.Unmarshal(body, &pubsubMsg); err != nil {
		return fmt.Errorf("failed to unmarshal Pub/Sub message: %w", err)
	}

	// Decode the base64-encoded data
	data, err := base64.StdEncoding.DecodeString(pubsubMsg.Message.Data)
	if err != nil {
		return fmt.Errorf("failed to decode message data: %w", err)
	}

	// Parse the JSON payload
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	clog.InfoContext(ctx, "Received message", "messageId", pubsubMsg.Message.MessageID, "dataLength", len(data))

	// Format the message using the template
	message, err := processMessage(payload, slackService.messageTemplate)
	if err != nil {
		return fmt.Errorf("failed to format message: %w", err)
	}

	// Send to Slack
	if err := slackService.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}

	clog.InfoContext(ctx, "Successfully sent message to Slack", "channel", slackService.channel)
	return nil
}

// SendMessage sends a message to the configured Slack channel
func (s *SlackService) SendMessage(ctx context.Context, message string) error {
	// Create a simple text message
	payload := map[string]interface{}{
		"channel": s.channel,
		"text":    message,
	}

	// Marshal the payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	// For webhook URLs, we need to make a direct HTTP POST
	resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to post to Slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Slack webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
