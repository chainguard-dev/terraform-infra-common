/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"cloud.google.com/go/compute/metadata"
	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
)

// State wraps a typed state with resource information.
type State[T any] struct {
	identity     string
	client       *sdk.GitHubClient
	resource     *Resource
	projectID    string
	serviceName  string
	stateManager *sdk.CommentStateManager
}

// NewState creates a new state instance for a specific resource.
func NewState[T any](identity string, client *sdk.GitHubClient, resource *Resource) *State[T] {
	// Get project ID once at creation
	projectID := getProjectID()

	// Get service name once at creation
	serviceName := os.Getenv("K_SERVICE")
	if serviceName == "" {
		serviceName = "unknown-service"
	}

	return &State[T]{
		identity:     identity,
		client:       client,
		resource:     resource,
		projectID:    projectID,
		serviceName:  serviceName,
		stateManager: sdk.NewCommentStateManager(identity),
	}
}

// Fetch retrieves the current state from the automation comment.
func (s *State[T]) Fetch(ctx context.Context) (*T, error) {
	log := clog.FromContext(ctx)

	// Find comment with our identity marker
	identityMarker := s.stateManager.GetIdentityMarker()
	comment, err := s.client.FindCommentByMarker(ctx, s.resource.Owner, s.resource.Repo, s.resource.Number, identityMarker)
	if err != nil {
		return nil, fmt.Errorf("failed to find comment: %w", err)
	}

	if comment == nil || comment.Body == nil {
		// No automation comment found
		return nil, nil
	}

	// Check if comment has state
	if !s.stateManager.HasState(*comment.Body) {
		// Comment exists but no state yet
		return nil, nil
	}

	// Extract state from comment
	var state T
	if err := s.stateManager.ExtractState(*comment.Body, &state); err != nil {
		log.Errorf("Failed to extract state from comment: %v", err)
		return nil, nil // Return nil state if we can't parse it
	}

	return &state, nil
}

// Commit updates or creates the automation comment with new state and message.
func (s *State[T]) Commit(ctx context.Context, state *T, message string) error {
	log := clog.FromContext(ctx)

	// Build the comment content with state
	content, err := s.stateManager.BuildCommentWithState(message, state, func() string {
		return s.buildBotInfoBlock()
	})
	if err != nil {
		return fmt.Errorf("failed to build comment content: %w", err)
	}

	// Find existing comment
	identityMarker := s.stateManager.GetIdentityMarker()
	existingComment, err := s.client.FindCommentByMarker(ctx, s.resource.Owner, s.resource.Repo, s.resource.Number, identityMarker)
	if err != nil {
		return fmt.Errorf("failed to find existing comment: %w", err)
	}

	if existingComment != nil {
		// Check if content has changed
		if existingComment.Body != nil && *existingComment.Body == content {
			log.Debug("Comment content unchanged, skipping update")
			return nil
		}
		log.With("comment_id", *existingComment.ID).Info("Updated automation comment with new state")
	} else {
		log.Info("Created automation comment with initial state")
	}

	// Use SDK's UpdateOrCreateComment to handle the update/create logic
	return s.client.UpdateOrCreateComment(ctx, s.resource.Owner, s.resource.Repo, s.resource.Number, identityMarker, content)
}

// buildBotInfoBlock creates an italicized block with bot info and logs link.
func (s *State[T]) buildBotInfoBlock() string {
	// Build the issue/PR URL
	issueURL := fmt.Sprintf("https://github.com/%s/%s/pull/%d", s.resource.Owner, s.resource.Repo, s.resource.Number)
	if s.resource.Type == ResourceTypeIssue {
		issueURL = fmt.Sprintf("https://github.com/%s/%s/issues/%d", s.resource.Owner, s.resource.Repo, s.resource.Number)
	}

	// URL encode the issue URL for the logs query
	encodedURL := url.QueryEscape(issueURL)

	// Build the Stackdriver logs URL
	logsURL := fmt.Sprintf(
		"https://console.cloud.google.com/logs/query;query=resource.type%%20%%3D%%20%%22cloud_run_revision%%22%%0Aresource.labels.service_name%%20%%3D%%20%%22%s%%22%%0AjsonPayload.key%%3D%%22%s%%22;storageScope=project;summaryFields=:false:32:beginning;duration=P2D?project=%s",
		s.serviceName,
		encodedURL,
		s.projectID,
	)

	return fmt.Sprintf("*🤖: %s ([logs](%s))*", s.identity, logsURL)
}

// getProjectID retrieves the GCP project ID.
func getProjectID() string {
	ctx := context.Background()
	projectID, err := metadata.ProjectIDWithContext(ctx)
	if err != nil {
		// Fallback to environment variable or default
		projectID = os.Getenv("GCP_PROJECT")
		if projectID == "" {
			projectID = "unknown-project"
		}
	}
	return projectID
}
