/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package statusmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/go-github/v75/github"
)

// Status represents the overall status of reconciliation
type Status[T any] struct {
	// ObservedGeneration is the last commit SHA that was fully processed
	ObservedGeneration string `json:"observedGeneration"`

	// Status represents the current status of the check run
	// Can be: "queued", "in_progress", or "completed"
	Status string `json:"status"`

	// Conclusion represents the conclusion when status is "completed"
	// Can be: "action_required", "cancelled", "failure", "neutral",
	// "success", "skipped", "stale", or "timed_out"
	Conclusion string `json:"conclusion,omitempty"`

	// Details contains reconciler-specific state data
	Details T `json:"details"`
}

// StatusManager manages reconciliation status via GitHub Check Runs
type StatusManager[T any] struct {
	identity    string
	projectID   string
	serviceName string
}

// NewStatusManager creates a new status manager with the given identity
func NewStatusManager[T any](ctx context.Context, identity string) (*StatusManager[T], error) {
	// Get project ID from metadata
	projectID, err := metadata.ProjectIDWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID from metadata: %w", err)
	}

	// Get service name once at startup
	serviceName, ok := os.LookupEnv("K_SERVICE")
	if !ok {
		return nil, fmt.Errorf("K_SERVICE environment variable not set")
	}

	return &StatusManager[T]{
		identity:    identity,
		projectID:   projectID,
		serviceName: serviceName,
	}, nil
}

// Session represents a reconciliation session for a specific PR
type Session[T any] struct {
	manager     *StatusManager[T]
	client      *github.Client
	owner       string
	repo        string
	sha         string
	prURL       string
	projectID   string
	serviceName string

	mu         sync.Mutex
	checkRunID *int64 // Set when we find an existing check run
}

// NewSession creates a new reconciliation session for a pull request
func (sm *StatusManager[T]) NewSession(client *github.Client, pr *github.PullRequest) *Session[T] {
	owner := pr.GetBase().GetRepo().GetOwner().GetLogin()
	repo := pr.GetBase().GetRepo().GetName()

	return &Session[T]{
		manager:     sm,
		client:      client,
		owner:       owner,
		repo:        repo,
		sha:         pr.GetHead().GetSHA(),
		prURL:       pr.GetHTMLURL(),
		projectID:   sm.projectID,
		serviceName: sm.serviceName,
	}
}

// getCheckRunID returns the stored check run ID if set
func (s *Session[T]) getCheckRunID() *int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checkRunID
}

// setCheckRunID stores a check run ID for future updates
func (s *Session[T]) setCheckRunID(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkRunID = &id
}

// buildDetailsURL builds the Cloud Logging URL with filters for this PR and SHA
func (s *Session[T]) buildDetailsURL() string {
	// Build the Cloud Logging URL with both PR and SHA filters
	// The query filters for:
	// - Cloud Run revision logs
	// - Specific service name
	// - The PR URL in jsonPayload.key
	// - The SHA in jsonPayload.sha
	query := fmt.Sprintf(`resource.type="cloud_run_revision"
resource.labels.service_name=%q
jsonPayload.key=%q
jsonPayload.sha=%q`,
		s.serviceName,
		s.prURL,
		s.sha,
	)

	encodedQuery := url.QueryEscape(query)

	return fmt.Sprintf(
		"https://console.cloud.google.com/logs/query;query=%s;storageScope=project;summaryFields=:false:32:beginning;duration=P2D?project=%s",
		encodedQuery,
		s.projectID,
	)
}

// ObservedState retrieves the last observed state for the current SHA
func (s *Session[T]) ObservedState(ctx context.Context) (*Status[T], error) {
	// Get check runs for this SHA
	checkRuns, _, err := s.client.Checks.ListCheckRunsForRef(
		ctx, s.owner, s.repo, s.sha,
		&github.ListCheckRunsOptions{
			CheckName: github.Ptr(s.manager.identity),
		})

	if err != nil {
		return nil, fmt.Errorf("listing check runs: %w", err)
	}

	// Find our check run
	for _, run := range checkRuns.CheckRuns {
		if run.GetName() == s.manager.identity {
			// Record the check run ID for potential updates
			s.setCheckRunID(run.GetID())

			// Extract status from output
			return extractStatusFromOutput[T](s.manager.identity, run.Output)
		}
	}

	return nil, nil // No status found
}

// ObservedStateAtSHA retrieves the status for a specific commit SHA without creating a session.
// This is useful for gathering historical status across multiple commits in a PR.
func (sm *StatusManager[T]) ObservedStateAtSHA(
	ctx context.Context,
	client *github.Client,
	pr *github.PullRequest,
	sha string,
) (*Status[T], error) {
	owner := pr.GetBase().GetRepo().GetOwner().GetLogin()
	repo := pr.GetBase().GetRepo().GetName()

	checkRuns, _, err := client.Checks.ListCheckRunsForRef(
		ctx, owner, repo, sha,
		&github.ListCheckRunsOptions{
			CheckName: github.Ptr(sm.identity),
		})

	if err != nil {
		return nil, fmt.Errorf("listing check runs: %w", err)
	}

	for _, run := range checkRuns.CheckRuns {
		if run.GetName() == sm.identity {
			return extractStatusFromOutput[T](sm.identity, run.Output)
		}
	}

	return nil, nil
}

// SetActualState updates the state for the current SHA
func (s *Session[T]) SetActualState(ctx context.Context, title string, status *Status[T]) error {
	// Ensure ObservedGeneration is set to current SHA
	status.ObservedGeneration = s.sha

	// Build markdown output with embedded JSON
	output, err := buildCheckRunOutput(s.manager.identity, status)
	if err != nil {
		return fmt.Errorf("building output: %w", err)
	}

	// Build the details URL for logs
	detailsURL := s.buildDetailsURL()

	// Only pass Conclusion if it's not empty
	var conclusionPtr *string
	if status.Conclusion != "" {
		conclusionPtr = &status.Conclusion
	}

	// Build CheckRunOutput with optional annotations
	checkOutput := &github.CheckRunOutput{
		Title:   github.Ptr(title),
		Summary: github.Ptr(output),
	}

	// Check if Details implements Annotated interface
	if annotated, ok := any(status.Details).(Annotated); ok {
		annotations := annotated.Annotations()
		if len(annotations) > 0 {
			checkOutput.Annotations = annotations
			checkOutput.AnnotationsCount = github.Ptr(len(annotations))
		}
	}

	// Check if we have a check run ID from ObservedState
	if checkRunID := s.getCheckRunID(); checkRunID != nil {
		// Update existing check run
		_, _, err = s.client.Checks.UpdateCheckRun(ctx, s.owner, s.repo, *checkRunID, github.UpdateCheckRunOptions{
			Name:       s.manager.identity,
			Status:     &status.Status,
			Conclusion: conclusionPtr,
			DetailsURL: &detailsURL,
			Output:     checkOutput,
		})

		if err != nil {
			return fmt.Errorf("updating check run: %w", err)
		}

		return nil
	}

	// Create new check run
	checkRun, _, err := s.client.Checks.CreateCheckRun(ctx, s.owner, s.repo, github.CreateCheckRunOptions{
		Name:       s.manager.identity,
		HeadSHA:    s.sha,
		Status:     &status.Status,
		Conclusion: conclusionPtr,
		DetailsURL: &detailsURL,
		Output:     checkOutput,
	})

	if err != nil {
		return fmt.Errorf("creating check run: %w", err)
	}

	// Store the ID for future updates
	s.setCheckRunID(checkRun.GetID())

	return nil
}

// markdownProvider is an interface for types that can provide markdown representation
type markdownProvider interface {
	Markdown() string
}

// Annotated is an interface for types that can provide GitHub check run annotations
type Annotated interface {
	Annotations() []*github.CheckRunAnnotation
}

// getStatusMarker returns the HTML comment marker for status data
func getStatusMarker(identity string) string {
	return fmt.Sprintf("<!--%s-status-->", identity)
}

// getStatusEndMarker returns the HTML comment end marker for status data
func getStatusEndMarker(identity string) string {
	return fmt.Sprintf("<!--/%s-status-->", identity)
}

// buildCheckRunOutput builds the markdown output with embedded status data
func buildCheckRunOutput[T any](identity string, status *Status[T]) (string, error) {
	var output strings.Builder

	// Check if Details implements Markdown() method
	if provider, ok := any(status.Details).(markdownProvider); ok {
		// Use the custom markdown representation
		markdown := provider.Markdown()
		if markdown != "" {
			output.WriteString(markdown)
		}
	}
	// If no Markdown() method or empty output, no visible content

	// Serialize status to JSON
	statusJSON, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling status: %w", err)
	}

	// Embed status data in HTML comments
	output.WriteString("\n\n")
	output.WriteString(getStatusMarker(identity))
	output.WriteString("\n<!--\n")
	output.WriteString(string(statusJSON))
	output.WriteString("\n-->\n")
	output.WriteString(getStatusEndMarker(identity))

	return output.String(), nil
}

// extractStatusFromOutput extracts the status JSON from check run output
func extractStatusFromOutput[T any](identity string, output *github.CheckRunOutput) (*Status[T], error) {
	if output == nil || output.Summary == nil {
		return nil, nil
	}

	body := *output.Summary
	statusMarker := getStatusMarker(identity)
	statusEndMarker := getStatusEndMarker(identity)

	// Find the status data between markers
	startIdx := strings.Index(body, statusMarker)
	if startIdx == -1 {
		return nil, nil
	}
	startIdx += len(statusMarker)

	endIdx := strings.Index(body[startIdx:], statusEndMarker)
	if endIdx == -1 {
		return nil, fmt.Errorf("malformed status: missing end marker")
	}

	statusJSON := strings.TrimSpace(body[startIdx : startIdx+endIdx])

	// Remove HTML comment wrapper
	statusJSON = strings.TrimPrefix(statusJSON, "<!--")
	statusJSON = strings.TrimSuffix(statusJSON, "-->")

	var status Status[T]
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status JSON: %w", err)
	}

	return &status, nil
}
