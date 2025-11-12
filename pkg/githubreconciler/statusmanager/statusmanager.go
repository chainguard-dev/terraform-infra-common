/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package statusmanager

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"

	"cloud.google.com/go/compute/metadata"
	"github.com/google/go-github/v75/github"

	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	internaltemplate "github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler/internal/template"
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
	identity         string
	projectID        string
	serviceName      string
	readOnly         bool
	templateExecutor *internaltemplate.Template[Status[T]]
}

// NewStatusManager creates a new status manager with the given identity
func NewStatusManager[T any](ctx context.Context, identity string) (*StatusManager[T], error) {
	return newStatusManager[T](ctx, identity, false)
}

// NewReadOnlyStatusManager creates a new read-only status manager with the given identity.
// A read-only status manager will fail any operations that attempt to mutate GitHub state.
func NewReadOnlyStatusManager[T any](ctx context.Context, identity string) (*StatusManager[T], error) {
	return newStatusManager[T](ctx, identity, true)
}

func newStatusManager[T any](ctx context.Context, identity string, readOnly bool) (*StatusManager[T], error) {
	// Get project ID from metadata
	projectID, err := metadata.ProjectIDWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID from metadata: %w", err)
	}

	// Get service name once at startup
	serviceName, ok := os.LookupEnv("K_SERVICE")
	if !ok {
		return nil, errors.New("K_SERVICE environment variable not set")
	}

	return &StatusManager[T]{
		identity:         identity,
		projectID:        projectID,
		serviceName:      serviceName,
		readOnly:         readOnly,
		templateExecutor: internaltemplate.New[Status[T]](identity, "-status", "status"),
	}, nil
}

// Session represents a reconciliation session for a specific resource
type Session[T any] struct {
	manager  *StatusManager[T]
	client   *github.Client
	resource *githubreconciler.Resource
	sha      string
	readOnly bool

	mu         sync.Mutex
	checkRunID *int64 // Set when we find an existing check run
}

// NewSession creates a new reconciliation session for a GitHub resource and SHA.
// The resource provides owner, repo, and URL (used as key for log filtering).
// The SHA is the commit to attach check runs to.
func (sm *StatusManager[T]) NewSession(client *github.Client, res *githubreconciler.Resource, sha string) *Session[T] {
	return &Session[T]{
		manager:  sm,
		client:   client,
		resource: res,
		sha:      sha,
		readOnly: sm.readOnly,
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

// buildDetailsURL builds the Cloud Logging URL with filters for this resource and SHA
func (s *Session[T]) buildDetailsURL() string {
	// Build the Cloud Logging URL with both key and SHA filters
	// The query filters for:
	// - Cloud Run revision logs
	// - Specific service name
	// - The resource key (PR URL) in jsonPayload.key
	// - The SHA in jsonPayload.sha
	query := fmt.Sprintf(`resource.type="cloud_run_revision"
resource.labels.service_name=%q
jsonPayload.key=%q
jsonPayload.sha=%q`,
		s.manager.serviceName,
		s.resource.URL,
		s.sha,
	)

	encodedQuery := url.QueryEscape(query)

	return fmt.Sprintf(
		"https://console.cloud.google.com/logs/query;query=%s;storageScope=project;summaryFields=:false:32:beginning;duration=P2D?project=%s",
		encodedQuery,
		s.manager.projectID,
	)
}

// ObservedState retrieves the last observed state for the current SHA
func (s *Session[T]) ObservedState(ctx context.Context) (*Status[T], error) {
	name, err := checkRunName(s.manager.identity, s.resource)
	if err != nil {
		return nil, err
	}

	// Get check runs for this SHA
	checkRuns, _, err := s.client.Checks.ListCheckRunsForRef(
		ctx, s.resource.Owner, s.resource.Repo, s.sha,
		&github.ListCheckRunsOptions{
			CheckName: github.Ptr(name),
		})

	if err != nil {
		return nil, fmt.Errorf("listing check runs: %w", err)
	}

	// Find our check run
	for _, run := range checkRuns.CheckRuns {
		if run.GetName() == name {
			// Record the check run ID for potential updates
			s.setCheckRunID(run.GetID())

			// Extract status from output
			return s.manager.extractStatusFromOutput(run.Output)
		}
	}

	return nil, nil // No status found
}

// ObservedStateAtSHA retrieves the status for a specific commit SHA without creating a session.
// This is useful for gathering historical status across multiple commits.
func (sm *StatusManager[T]) ObservedStateAtSHA(
	ctx context.Context,
	client *github.Client,
	res *githubreconciler.Resource,
	sha string,
) (*Status[T], error) {
	name, err := checkRunName(sm.identity, res)
	if err != nil {
		return nil, err
	}

	checkRuns, _, err := client.Checks.ListCheckRunsForRef(
		ctx, res.Owner, res.Repo, sha,
		&github.ListCheckRunsOptions{
			CheckName: github.Ptr(name),
		})

	if err != nil {
		return nil, fmt.Errorf("listing check runs: %w", err)
	}

	for _, run := range checkRuns.CheckRuns {
		if run.GetName() == name {
			return sm.extractStatusFromOutput(run.Output)
		}
	}

	return nil, nil
}

// SetActualState updates the state for the current SHA
func (s *Session[T]) SetActualState(ctx context.Context, title string, status *Status[T]) error {
	if s.readOnly {
		return errors.New("cannot set actual state: status manager is read-only")
	}

	name, err := checkRunName(s.manager.identity, s.resource)
	if err != nil {
		return err
	}

	// Ensure ObservedGeneration is set to current SHA
	status.ObservedGeneration = s.sha

	// Build markdown output with embedded JSON
	output, err := s.manager.buildCheckRunOutput(status)
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
		_, _, err = s.client.Checks.UpdateCheckRun(ctx, s.resource.Owner, s.resource.Repo, *checkRunID, github.UpdateCheckRunOptions{
			Name:       name,
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
	checkRun, _, err := s.client.Checks.CreateCheckRun(ctx, s.resource.Owner, s.resource.Repo, github.CreateCheckRunOptions{
		Name:       name,
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

// checkRunName returns the check run name for the given identity and resource.
// For pull requests, returns the identity as-is.
// For paths, returns "{identity} ({path})" to distinguish different paths.
// Returns an error if the resource type is not supported by StatusManager.
func checkRunName(identity string, res *githubreconciler.Resource) (string, error) {
	switch res.Type {
	case githubreconciler.ResourceTypePullRequest:
		return identity, nil
	case githubreconciler.ResourceTypePath:
		return fmt.Sprintf("%s (%s)", identity, res.Path), nil
	case githubreconciler.ResourceTypeIssue:
		return "", errors.New("issues are not supported by StatusManager")
	default:
		return "", fmt.Errorf("unrecognized resource type: %s", res.Type)
	}
}

// buildCheckRunOutput builds the markdown output with embedded status data
func (sm *StatusManager[T]) buildCheckRunOutput(status *Status[T]) (string, error) {
	var markdown string

	// Check if Details implements Markdown() method
	if provider, ok := any(status.Details).(markdownProvider); ok {
		// Use the custom markdown representation
		markdown = provider.Markdown()
	}
	// If no Markdown() method or empty output, no visible content

	// Embed status data using the template executor
	return sm.templateExecutor.Embed(markdown, status)
}

// extractStatusFromOutput extracts the status JSON from check run output
func (sm *StatusManager[T]) extractStatusFromOutput(output *github.CheckRunOutput) (*Status[T], error) {
	if output == nil || output.Summary == nil {
		return nil, nil
	}

	// Extract status data using the template executor
	return sm.templateExecutor.Extract(*output.Summary)
}
