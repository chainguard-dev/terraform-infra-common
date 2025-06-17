/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chainguard-dev/clog"
	"github.com/google/go-github/v72/github"
)

// StateManager manages reconciler state stored in GitHub comments.
type StateManager struct {
	identity string
}

// NewStateManager creates a new state manager with the given identity.
// The identity is used to identify comments created by this reconciler.
func NewStateManager(identity string) *StateManager {
	return &StateManager{
		identity: identity,
	}
}

// Identity returns the identity used by this state manager.
func (sm *StateManager) Identity() string {
	return sm.identity
}

// State wraps a typed state with resource information.
type State[T any] struct {
	identity string
	client   *github.Client
	resource *Resource
}

// NewState creates a new state instance for a specific resource.
func NewState[T any](identity string, client *github.Client, resource *Resource) *State[T] {
	return &State[T]{
		identity: identity,
		client:   client,
		resource: resource,
	}
}

// getStateMarker returns the HTML comment marker for state data.
func (s *State[T]) getStateMarker() string {
	return fmt.Sprintf("<!--%s-state-->", s.identity)
}

// getIdentityMarker returns the HTML comment marker for the identity.
func (s *State[T]) getIdentityMarker() string {
	return fmt.Sprintf("<!--%s-->", s.identity)
}

// Fetch retrieves the current state from the automation comment.
func (s *State[T]) Fetch(ctx context.Context) (*T, error) {
	log := clog.FromContext(ctx)

	// List all comments to find our automation comment
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	identityMarker := s.getIdentityMarker()
	stateMarker := s.getStateMarker()

	for {
		comments, resp, err := s.client.Issues.ListComments(ctx, s.resource.Owner, s.resource.Repo, s.resource.Number, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list comments: %w", err)
		}

		for _, comment := range comments {
			if comment.Body != nil && strings.Contains(*comment.Body, identityMarker) {
				// Found our comment, extract state
				if strings.Contains(*comment.Body, stateMarker) {
					state, err := s.extractState(*comment.Body)
					if err != nil {
						log.Errorf("Failed to extract state from comment: %v", err)
						return nil, nil // Return nil state if we can't parse it
					}
					return state, nil
				}
				// Comment exists but no state yet
				return nil, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// No automation comment found
	return nil, nil
}

// extractState extracts the state JSON from the comment body.
func (s *State[T]) extractState(body string) (*T, error) {
	stateMarker := s.getStateMarker()
	stateEndMarker := fmt.Sprintf("<!--/%s-state-->", s.identity)

	// Find the state data between markers
	startIdx := strings.Index(body, stateMarker)
	if startIdx == -1 {
		return nil, nil
	}
	startIdx += len(stateMarker)

	endIdx := strings.Index(body[startIdx:], stateEndMarker)
	if endIdx == -1 {
		return nil, fmt.Errorf("malformed state: missing end marker")
	}

	stateJSON := strings.TrimSpace(body[startIdx : startIdx+endIdx])

	// Remove HTML comment wrapper if present
	stateJSON = strings.TrimPrefix(stateJSON, "<!--")
	stateJSON = strings.TrimSuffix(stateJSON, "-->")
	stateJSON = strings.TrimSpace(stateJSON)

	var state T
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// Commit updates or creates the automation comment with new state and message.
func (s *State[T]) Commit(ctx context.Context, state *T, message string) error {
	log := clog.FromContext(ctx)

	// Serialize the state
	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Build the comment content
	content := s.buildCommentContent(string(stateJSON), message)

	// Find existing comment
	existingComment, err := s.findExistingComment(ctx)
	if err != nil {
		return fmt.Errorf("failed to find existing comment: %w", err)
	}

	if existingComment != nil {
		// Check if content has changed
		if existingComment.Body != nil && *existingComment.Body == content {
			log.Debug("Comment content unchanged, skipping update")
			return nil
		}

		// Update existing comment
		updatedComment := &github.IssueComment{
			Body: &content,
		}

		_, _, err = s.client.Issues.EditComment(ctx, s.resource.Owner, s.resource.Repo, *existingComment.ID, updatedComment)
		if err != nil {
			return fmt.Errorf("failed to update comment: %w", err)
		}

		log.With("comment_id", *existingComment.ID).Info("Updated automation comment with new state")
	} else {
		// Create new comment
		newComment := &github.IssueComment{
			Body: &content,
		}

		_, _, err = s.client.Issues.CreateComment(ctx, s.resource.Owner, s.resource.Repo, s.resource.Number, newComment)
		if err != nil {
			return fmt.Errorf("failed to create comment: %w", err)
		}

		log.Info("Created automation comment with initial state")
	}

	return nil
}

// findExistingComment finds the automation comment if it exists.
func (s *State[T]) findExistingComment(ctx context.Context) (*github.IssueComment, error) {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	identityMarker := s.getIdentityMarker()

	for {
		comments, resp, err := s.client.Issues.ListComments(ctx, s.resource.Owner, s.resource.Repo, s.resource.Number, opts)
		if err != nil {
			return nil, err
		}

		for _, comment := range comments {
			if comment.Body != nil && strings.Contains(*comment.Body, identityMarker) {
				return comment, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil, nil
}

// buildCommentContent builds the comment content with state and message.
func (s *State[T]) buildCommentContent(stateJSON, message string) string {
	var content strings.Builder

	// Identity marker
	content.WriteString(s.getIdentityMarker())
	content.WriteString("\n\n")

	// User-visible message
	content.WriteString(message)
	content.WriteString("\n\n")

	// State data (hidden in HTML comment)
	content.WriteString(s.getStateMarker())
	content.WriteString("\n<!--\n")
	content.WriteString(stateJSON)
	content.WriteString("\n-->\n")
	content.WriteString(fmt.Sprintf("<!--/%s-state-->", s.identity))

	return content.String()
}
