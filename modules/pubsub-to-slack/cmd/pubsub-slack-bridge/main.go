/*
Copyright 2024 Chainguard, Inc.
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
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/chainguard-dev/clog"
	"github.com/sethvargo/go-envconfig"
)

// Config holds the application configuration
type Config struct {
	Port               string `env:"PORT,default=8080"`
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
	webhookURL string
	channel    string
	template   string
}

func main() {
	ctx := context.Background()
	logger := clog.New(slog.Default().Handler())

	var config Config
	if err := envconfig.Process(ctx, &config); err != nil {
		logger.Fatalf("failed to process config: %v", err)
	}

	// Get Slack webhook URL from Secret Manager
	webhookURL, err := getSecretValue(ctx, config.ProjectID, config.SlackWebhookSecret)
	if err != nil {
		logger.Fatalf("failed to get Slack webhook URL: %v", err)
	}

	// Initialize Slack service
	slackService := &SlackService{
		webhookURL: webhookURL,
		channel:    config.SlackChannel,
		template:   config.MessageTemplate,
	}

	// Set up HTTP handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if err := handlePubSubMessage(ctx, logger, slackService, w, r); err != nil {
			logger.Errorf("failed to handle Pub/Sub message: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	logger.Infof("Starting server on port %s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, nil); err != nil {
		logger.Fatalf("server failed: %v", err)
	}
}

// getSecretValue retrieves a secret value from Google Secret Manager
func getSecretValue(ctx context.Context, projectID, secretID string) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secretmanager client: %v", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID),
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret version: %v", err)
	}

	return string(result.Payload.Data), nil
}

// handlePubSubMessage processes incoming Pub/Sub push messages
func handlePubSubMessage(ctx context.Context, logger *clog.Logger, slackService *SlackService, w http.ResponseWriter, r *http.Request) error {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %v", err)
	}

	// Parse the Pub/Sub message
	var pubsubMsg PubSubMessage
	if err := json.Unmarshal(body, &pubsubMsg); err != nil {
		return fmt.Errorf("failed to unmarshal Pub/Sub message: %v", err)
	}

	// Decode the base64-encoded data
	data, err := base64.StdEncoding.DecodeString(pubsubMsg.Message.Data)
	if err != nil {
		return fmt.Errorf("failed to decode message data: %v", err)
	}

	// Parse the JSON payload
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %v", err)
	}

	logger.Infof("Received message: %s", string(data))

	// Format the message using the template
	message, err := formatMessage(slackService.template, payload)
	if err != nil {
		return fmt.Errorf("failed to format message: %v", err)
	}

	// Send to Slack
	if err := slackService.SendMessage(ctx, message); err != nil {
		return fmt.Errorf("failed to send Slack message: %v", err)
	}

	logger.Infof("Successfully sent message to Slack channel %s", slackService.channel)
	return nil
}

// formatMessage applies the template to the payload data
func formatMessage(template string, payload map[string]interface{}) (string, error) {
	result := template

	// Simple template substitution using ${field} syntax
	// This could be enhanced with more sophisticated templating if needed
	for key, value := range payload {
		placeholder := fmt.Sprintf("${%s}", key)

		// Convert value to string
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		case float64:
			// Handle numbers (JSON unmarshals numbers as float64)
			valueStr = fmt.Sprintf("%.2f", v)
		case bool:
			valueStr = fmt.Sprintf("%t", v)
		case nil:
			valueStr = "null"
		default:
			// For complex objects, marshal back to JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				valueStr = fmt.Sprintf("%v", v)
			} else {
				valueStr = string(jsonBytes)
			}
		}

		result = strings.ReplaceAll(result, placeholder, valueStr)
	}

	return result, nil
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
		return fmt.Errorf("failed to marshal Slack payload: %v", err)
	}

	// For webhook URLs, we need to make a direct HTTP POST
	resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to post to Slack webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Slack webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
