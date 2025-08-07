/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CommentStateManager helps manage JSON state embedded in GitHub comments.
type CommentStateManager struct {
	Identity string
}

// NewCommentStateManager creates a new state manager with the given identity.
func NewCommentStateManager(identity string) *CommentStateManager {
	return &CommentStateManager{
		Identity: identity,
	}
}

// GetStateMarker returns the HTML comment marker for state data.
func (sm *CommentStateManager) GetStateMarker() string {
	return fmt.Sprintf("<!--%s-state-->", sm.Identity)
}

// GetStateEndMarker returns the HTML comment end marker for state data.
func (sm *CommentStateManager) GetStateEndMarker() string {
	return fmt.Sprintf("<!--/%s-state-->", sm.Identity)
}

// GetIdentityMarker returns the HTML comment marker for the identity.
func (sm *CommentStateManager) GetIdentityMarker() string {
	return fmt.Sprintf("<!--%s-->", sm.Identity)
}

// EmbedState embeds JSON state data in a comment string.
// The state is hidden in HTML comments between state markers.
func (sm *CommentStateManager) EmbedState(message string, state interface{}) (string, error) {
	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %w", err)
	}

	var content strings.Builder

	// Identity marker
	content.WriteString(sm.GetIdentityMarker())
	content.WriteString("\n\n")

	// User-visible message
	content.WriteString(message)
	content.WriteString("\n\n")

	// State data (hidden in HTML comment)
	content.WriteString(sm.GetStateMarker())
	content.WriteString("\n<!--\n")
	content.WriteString(string(stateJSON))
	content.WriteString("\n-->\n")
	content.WriteString(sm.GetStateEndMarker())

	return content.String(), nil
}

// ExtractState extracts JSON state from a comment body.
// Returns nil if no state is found.
func (sm *CommentStateManager) ExtractState(body string, state interface{}) error {
	stateMarker := sm.GetStateMarker()
	stateEndMarker := sm.GetStateEndMarker()

	// Find the state data between markers
	startIdx := strings.Index(body, stateMarker)
	if startIdx == -1 {
		return fmt.Errorf("state marker not found")
	}
	startIdx += len(stateMarker)

	endIdx := strings.Index(body[startIdx:], stateEndMarker)
	if endIdx == -1 {
		return fmt.Errorf("malformed state: missing end marker")
	}

	stateJSON := strings.TrimSpace(body[startIdx : startIdx+endIdx])

	// Remove HTML comment wrapper if present
	stateJSON = strings.TrimPrefix(stateJSON, "<!--")
	stateJSON = strings.TrimSuffix(stateJSON, "-->")
	stateJSON = strings.TrimSpace(stateJSON)

	if err := json.Unmarshal([]byte(stateJSON), state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return nil
}

// BuildCommentWithState builds a comment with an identity marker, message, and embedded state.
// This combines the identity marker, user message, and hidden state into a single comment.
func (sm *CommentStateManager) BuildCommentWithState(message string, state interface{}, headerFunc func() string) (string, error) {
	stateJSON, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %w", err)
	}

	var content strings.Builder

	// Identity marker
	content.WriteString(sm.GetIdentityMarker())
	content.WriteString("\n\n")

	// Optional header (e.g., bot info)
	if headerFunc != nil {
		header := headerFunc()
		if header != "" {
			content.WriteString(header)
			content.WriteString("\n\n---\n\n")
		}
	}

	// User-visible message
	content.WriteString(message)
	content.WriteString("\n\n")

	// State data (hidden in HTML comment)
	content.WriteString(sm.GetStateMarker())
	content.WriteString("\n<!--\n")
	content.WriteString(string(stateJSON))
	content.WriteString("\n-->\n")
	content.WriteString(sm.GetStateEndMarker())

	return content.String(), nil
}

// HasState checks if a comment body contains state for this identity.
func (sm *CommentStateManager) HasState(body string) bool {
	return strings.Contains(body, sm.GetStateMarker()) && strings.Contains(body, sm.GetStateEndMarker())
}

