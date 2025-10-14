/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package statusmanager

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v75/github"

	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
)

// TestDetails is a test implementation of Details
type TestDetails struct {
	Message string   `json:"message"`
	Count   int      `json:"count"`
	Items   []string `json:"items,omitempty"`
}

// TestDetailsWithMarkdown includes a Markdown method
type TestDetailsWithMarkdown struct {
	Message string   `json:"message"`
	Count   int      `json:"count"`
	Items   []string `json:"items,omitempty"`
}

func (d TestDetailsWithMarkdown) Markdown() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**Message**: %s\n", d.Message))
	b.WriteString(fmt.Sprintf("**Count**: %d\n", d.Count))
	if len(d.Items) > 0 {
		b.WriteString("**Items**:\n")
		for _, item := range d.Items {
			b.WriteString(fmt.Sprintf("- %s\n", item))
		}
	}
	return b.String()
}

func TestGetStatusMarker(t *testing.T) {
	tests := []struct {
		name     string
		identity string
		want     string
	}{{
		name:     "simple identity",
		identity: "autofix",
		want:     "<!--autofix-status-->",
	}, {
		name:     "identity with spaces",
		identity: "my reconciler",
		want:     "<!--my reconciler-status-->",
	}, {
		name:     "identity with special chars",
		identity: "test-reconciler_v1.0",
		want:     "<!--test-reconciler_v1.0-status-->",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusMarker(tt.identity)
			if got != tt.want {
				t.Errorf("getStatusMarker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStatusEndMarker(t *testing.T) {
	tests := []struct {
		name     string
		identity string
		want     string
	}{{
		name:     "simple identity",
		identity: "autofix",
		want:     "<!--/autofix-status-->",
	}, {
		name:     "identity with spaces",
		identity: "my reconciler",
		want:     "<!--/my reconciler-status-->",
	}, {
		name:     "identity with special chars",
		identity: "test-reconciler_v1.0",
		want:     "<!--/test-reconciler_v1.0-status-->",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getStatusEndMarker(tt.identity)
			if got != tt.want {
				t.Errorf("getStatusEndMarker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundtripWithoutMarkdown(t *testing.T) {
	identity := "test-reconciler"

	tests := []struct {
		name   string
		status Status[TestDetails]
	}{{
		name: "completed with success",
		status: Status[TestDetails]{
			ObservedGeneration: "abc123def456",
			Status:             "completed",
			Conclusion:         "success",
			Details: TestDetails{
				Message: "Processing complete",
				Count:   42,
				Items:   []string{"item1", "item2", "item3"},
			},
		},
	}, {
		name: "in progress without conclusion",
		status: Status[TestDetails]{
			ObservedGeneration: "def789ghi012",
			Status:             "in_progress",
			Details: TestDetails{
				Message: "Still processing",
				Count:   10,
			},
		},
	}, {
		name: "queued state",
		status: Status[TestDetails]{
			ObservedGeneration: "xyz789abc123",
			Status:             "queued",
			Details: TestDetails{
				Message: "Waiting to start",
				Count:   0,
			},
		},
	}, {
		name: "completed with failure",
		status: Status[TestDetails]{
			ObservedGeneration: "fail123xyz789",
			Status:             "completed",
			Conclusion:         "failure",
			Details: TestDetails{
				Message: "Processing failed",
				Count:   -1,
				Items:   []string{"error1", "error2"},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the output
			output, err := buildCheckRunOutput(identity, &tt.status)
			if err != nil {
				t.Fatalf("buildCheckRunOutput() error = %v", err)
			}

			// Verify markers are present
			if !strings.Contains(output, getStatusMarker(identity)) {
				t.Errorf("Output missing start marker")
			}
			if !strings.Contains(output, getStatusEndMarker(identity)) {
				t.Errorf("Output missing end marker")
			}

			// Since TestDetails doesn't implement Markdown(), output should start with markers
			if !strings.HasPrefix(strings.TrimSpace(output), getStatusMarker(identity)) {
				t.Errorf("Output should start with marker when no Markdown() method")
			}

			// Extract the status back
			checkRunOutput := &github.CheckRunOutput{
				Summary: github.Ptr(output),
			}

			extracted, err := extractStatusFromOutput[TestDetails](identity, checkRunOutput)
			if err != nil {
				t.Fatalf("extractStatusFromOutput() error = %v", err)
			}

			if extracted == nil {
				t.Fatal("extractStatusFromOutput() returned nil")
				return
			}

			// Compare the extracted status with original
			if diff := cmp.Diff(tt.status, *extracted); diff != "" {
				t.Errorf("Roundtrip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRoundtripWithMarkdown(t *testing.T) {
	identity := "markdown-reconciler"

	status := Status[TestDetailsWithMarkdown]{
		ObservedGeneration: "abc123def456",
		Status:             "completed",
		Conclusion:         "success",
		Details: TestDetailsWithMarkdown{
			Message: "All tests passed",
			Count:   100,
			Items:   []string{"test1", "test2", "test3"},
		},
	}

	// Build the output
	output, err := buildCheckRunOutput(identity, &status)
	if err != nil {
		t.Fatalf("buildCheckRunOutput() error = %v", err)
	}

	// Verify markdown content is present
	expectedMarkdown := status.Details.Markdown()
	if !strings.Contains(output, expectedMarkdown) {
		t.Errorf("Output missing markdown content")
	}

	// Verify the markdown comes before the markers
	markdownIndex := strings.Index(output, expectedMarkdown)
	markerIndex := strings.Index(output, getStatusMarker(identity))
	if markdownIndex > markerIndex {
		t.Errorf("Markdown should appear before status markers")
	}

	// Extract the status back
	checkRunOutput := &github.CheckRunOutput{
		Summary: github.Ptr(output),
	}

	extracted, err := extractStatusFromOutput[TestDetailsWithMarkdown](identity, checkRunOutput)
	if err != nil {
		t.Fatalf("extractStatusFromOutput() error = %v", err)
	}

	if extracted == nil {
		t.Fatal("extractStatusFromOutput() returned nil")
		return
	}

	// Verify roundtrip succeeded
	if diff := cmp.Diff(status, *extracted); diff != "" {
		t.Errorf("Roundtrip with markdown mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractStatusFromOutputEdgeCases(t *testing.T) {
	identity := "test-reconciler"

	tests := []struct {
		name    string
		output  *github.CheckRunOutput
		wantNil bool
		wantErr bool
	}{{
		name:    "nil output",
		output:  nil,
		wantNil: true,
	}, {
		name: "nil summary",
		output: &github.CheckRunOutput{
			Title: github.Ptr("Test"),
		},
		wantNil: true,
	}, {
		name: "empty summary",
		output: &github.CheckRunOutput{
			Summary: github.Ptr(""),
		},
		wantNil: true,
	}, {
		name: "no markers in summary",
		output: &github.CheckRunOutput{
			Summary: github.Ptr("Just some random text without markers"),
		},
		wantNil: true,
	}, {
		name: "only start marker",
		output: &github.CheckRunOutput{
			Summary: github.Ptr(fmt.Sprintf("%s\n<!--\n{}\n", getStatusMarker(identity))),
		},
		wantNil: true, // Should return nil
		wantErr: true, // With error for missing end marker
	}, {
		name: "malformed JSON",
		output: &github.CheckRunOutput{
			Summary: github.Ptr(fmt.Sprintf("%s\n<!--\n{invalid json}\n-->\n%s",
				getStatusMarker(identity), getStatusEndMarker(identity))),
		},
		wantNil: true, // Returns nil when JSON unmarshal fails
		wantErr: true, // And returns error
	}, {
		name: "valid but empty JSON",
		output: &github.CheckRunOutput{
			Summary: github.Ptr(fmt.Sprintf("%s\n<!--\n{}\n-->\n%s",
				getStatusMarker(identity), getStatusEndMarker(identity))),
		},
		wantNil: false,
		wantErr: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractStatusFromOutput[TestDetails](identity, tt.output)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractStatusFromOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (got == nil) != tt.wantNil {
				t.Errorf("extractStatusFromOutput() = %v, wantNil %v", got, tt.wantNil)
			}
		})
	}
}

func TestComplexRoundtrip(t *testing.T) {
	// Test with special characters and escaping
	identity := "test-reconciler"

	status := Status[TestDetails]{
		ObservedGeneration: "sha-with-special-chars!@#$%",
		Status:             "completed",
		Conclusion:         "success",
		Details: TestDetails{
			Message: "Message with \"quotes\" and 'apostrophes' and\nnewlines",
			Count:   999,
			Items:   []string{"<html>", "<!--comment-->", `backslash\test`},
		},
	}

	// Build and extract
	output, err := buildCheckRunOutput(identity, &status)
	if err != nil {
		t.Fatalf("buildCheckRunOutput() error = %v", err)
	}

	checkRunOutput := &github.CheckRunOutput{
		Summary: github.Ptr(output),
	}

	extracted, err := extractStatusFromOutput[TestDetails](identity, checkRunOutput)
	if err != nil {
		t.Fatalf("extractStatusFromOutput() error = %v", err)
	}

	if extracted == nil {
		t.Fatal("extractStatusFromOutput() returned nil")
		return
	}

	// Verify special characters survived the roundtrip
	if diff := cmp.Diff(status, *extracted); diff != "" {
		t.Errorf("Complex roundtrip mismatch (-want +got):\n%s", diff)
	}
}

func TestNewSession(t *testing.T) {
	// Create a StatusManager
	sm := &StatusManager[TestDetails]{
		identity:    "test-autofix",
		projectID:   "my-gcp-project",
		serviceName: "autofix-service",
	}

	tests := []struct {
		name            string
		pr              *github.PullRequest
		expectedOwner   string
		expectedRepo    string
		expectedSHA     string
		expectedPRURL   string
		expectedProject string
		expectedService string
	}{{
		name: "standard pull request",
		pr: &github.PullRequest{
			Number:  github.Ptr(123),
			URL:     github.Ptr("https://api.github.com/repos/owner/repo/pulls/123"),
			HTMLURL: github.Ptr("https://github.com/owner/repo/pull/123"),
			Base: &github.PullRequestBranch{
				Repo: &github.Repository{
					Owner: &github.User{
						Login: github.Ptr("test-owner"),
					},
					Name: github.Ptr("test-repo"),
				},
			},
			Head: &github.PullRequestBranch{
				SHA: github.Ptr("abc123def456"),
			},
		},
		expectedOwner:   "test-owner",
		expectedRepo:    "test-repo",
		expectedSHA:     "abc123def456",
		expectedPRURL:   "https://github.com/owner/repo/pull/123",
		expectedProject: "my-gcp-project",
		expectedService: "autofix-service",
	}, {
		name: "pull request with special characters",
		pr: &github.PullRequest{
			Number:  github.Ptr(456),
			URL:     github.Ptr("https://api.github.com/repos/org-with-dash/repo_with_underscore/pulls/456"),
			HTMLURL: github.Ptr("https://github.com/org-with-dash/repo_with_underscore/pull/456"),
			Base: &github.PullRequestBranch{
				Repo: &github.Repository{
					Owner: &github.User{
						Login: github.Ptr("org-with-dash"),
					},
					Name: github.Ptr("repo_with_underscore"),
				},
			},
			Head: &github.PullRequestBranch{
				SHA: github.Ptr("xyz789abc123"),
			},
		},
		expectedOwner:   "org-with-dash",
		expectedRepo:    "repo_with_underscore",
		expectedSHA:     "xyz789abc123",
		expectedPRURL:   "https://github.com/org-with-dash/repo_with_underscore/pull/456",
		expectedProject: "my-gcp-project",
		expectedService: "autofix-service",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock GitHub client
			client := &github.Client{}

			// Create a Resource from the PR
			owner := tt.pr.GetBase().GetRepo().GetOwner().GetLogin()
			repo := tt.pr.GetBase().GetRepo().GetName()
			res := &githubreconciler.Resource{
				Owner: owner,
				Repo:  repo,
				URL:   tt.pr.GetHTMLURL(),
			}

			session := sm.NewSession(client, res, tt.pr.GetHead().GetSHA())

			// Verify session fields
			if session.resource.Owner != tt.expectedOwner {
				t.Errorf("resource.Owner = %q, want %q", session.resource.Owner, tt.expectedOwner)
			}
			if session.resource.Repo != tt.expectedRepo {
				t.Errorf("resource.Repo = %q, want %q", session.resource.Repo, tt.expectedRepo)
			}
			if session.sha != tt.expectedSHA {
				t.Errorf("sha = %q, want %q", session.sha, tt.expectedSHA)
			}
			if session.resource.URL != tt.expectedPRURL {
				t.Errorf("resource.URL = %q, want %q", session.resource.URL, tt.expectedPRURL)
			}
			if session.manager.projectID != tt.expectedProject {
				t.Errorf("manager.projectID = %q, want %q", session.manager.projectID, tt.expectedProject)
			}
			if session.manager.serviceName != tt.expectedService {
				t.Errorf("manager.serviceName = %q, want %q", session.manager.serviceName, tt.expectedService)
			}

			// Verify manager reference
			if session.manager != sm {
				t.Error("session.manager should reference the StatusManager")
			}

			// Verify client reference
			if session.client != client {
				t.Error("session.client should reference the GitHub client")
			}

			// Verify checkRunID is initially nil
			if session.checkRunID != nil {
				t.Error("checkRunID should be initially nil")
			}
		})
	}
}

func TestCheckRunNameValidation(t *testing.T) {
	tests := []struct {
		name        string
		resource    *githubreconciler.Resource
		wantErr     bool
		errContains string
		wantName    string
	}{{
		name: "pull request is supported",
		resource: &githubreconciler.Resource{
			Owner: "test-owner",
			Repo:  "test-repo",
			Type:  githubreconciler.ResourceTypePullRequest,
			URL:   "https://github.com/test-owner/test-repo/pull/123",
		},
		wantErr:  false,
		wantName: "test-identity",
	}, {
		name: "path is supported",
		resource: &githubreconciler.Resource{
			Owner: "test-owner",
			Repo:  "test-repo",
			Type:  githubreconciler.ResourceTypePath,
			Path:  "pkg/foo/bar.go",
			URL:   "https://github.com/test-owner/test-repo/blob/main/pkg/foo/bar.go",
		},
		wantErr:  false,
		wantName: "test-identity (pkg/foo/bar.go)",
	}, {
		name: "issue is not supported",
		resource: &githubreconciler.Resource{
			Owner: "test-owner",
			Repo:  "test-repo",
			Type:  githubreconciler.ResourceTypeIssue,
			URL:   "https://github.com/test-owner/test-repo/issues/123",
		},
		wantErr:     true,
		errContains: "issues are not supported by StatusManager",
	}, {
		name: "unrecognized type",
		resource: &githubreconciler.Resource{
			Owner: "test-owner",
			Repo:  "test-repo",
			Type:  githubreconciler.ResourceType("unknown"),
			URL:   "https://github.com/test-owner/test-repo",
		},
		wantErr:     true,
		errContains: "unrecognized resource type",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, err := checkRunName("test-identity", tt.resource)

			if tt.wantErr {
				if err == nil {
					t.Error("checkRunName() error = nil, wantErr = true")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("checkRunName() error = %v, want error containing %q", err, tt.errContains)
				}
				if name != "" {
					t.Errorf("checkRunName() name = %q, want empty string on error", name)
				}
			} else {
				if err != nil {
					t.Errorf("checkRunName() error = %v, wantErr = false", err)
				}
				if name != tt.wantName {
					t.Errorf("checkRunName() name = %q, want = %q", name, tt.wantName)
				}
			}
		})
	}
}

func TestGetSetCheckRunID(t *testing.T) {
	// Create a test session manually
	session := &Session[TestDetails]{
		manager: &StatusManager[TestDetails]{
			identity:    "test-reconciler",
			projectID:   "test-project",
			serviceName: "test-service",
		},
		resource: &githubreconciler.Resource{
			Owner: "test-owner",
			Repo:  "test-repo",
			URL:   "https://github.com/test-owner/test-repo/pull/42",
		},
		sha: "abc123",
	}

	// Initially should be nil
	if id := session.getCheckRunID(); id != nil {
		t.Errorf("Initial checkRunID should be nil, got %v", id)
	}

	// Set a check run ID
	var testID int64 = 12345
	session.setCheckRunID(testID)

	// Should retrieve the same ID
	if id := session.getCheckRunID(); id == nil {
		t.Error("checkRunID should not be nil after setting")
	} else if *id != testID {
		t.Errorf("checkRunID = %d, want %d", *id, testID)
	}

	// Test concurrent access (basic race condition test)
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			session.setCheckRunID(int64(i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = session.getCheckRunID()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Final value should be set
	if id := session.getCheckRunID(); id == nil {
		t.Error("checkRunID should not be nil after concurrent access")
	}
}

func TestBuildDetailsURL(t *testing.T) {
	tests := []struct {
		name         string
		session      *Session[TestDetails]
		wantContains []string
	}{{
		name: "standard session",
		session: &Session[TestDetails]{
			manager: &StatusManager[TestDetails]{
				identity:    "test-reconciler",
				projectID:   "my-project",
				serviceName: "autofix-service",
			},
			resource: &githubreconciler.Resource{
				Owner: "chainguard-dev",
				Repo:  "mono",
				URL:   "https://github.com/chainguard-dev/mono/pull/123",
			},
			sha: "def456ghi789",
		},
		wantContains: []string{
			"console.cloud.google.com/logs/query",
			"project=my-project",
			"resource.type%3D%22cloud_run_revision%22",
			"resource.labels.service_name%3D%22autofix-service%22",
			"jsonPayload.key%3D%22https%3A%2F%2Fgithub.com%2Fchainguard-dev%2Fmono%2Fpull%2F123%22",
			"jsonPayload.sha%3D%22def456ghi789%22",
		},
	}, {
		name: "session with special characters",
		session: &Session[TestDetails]{
			manager: &StatusManager[TestDetails]{
				identity:    "test-reconciler",
				projectID:   "project-with-dash",
				serviceName: "service_with_underscore",
			},
			resource: &githubreconciler.Resource{
				Owner: "test-org",
				Repo:  "test-repo",
				URL:   "https://github.com/test-org/test-repo/pull/456",
			},
			sha: "sha+with+plus",
		},
		wantContains: []string{
			"project=project-with-dash",
			"resource.labels.service_name%3D%22service_with_underscore%22",
			"jsonPayload.sha%3D%22sha%2Bwith%2Bplus%22",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.session.buildDetailsURL()

			// Check that URL contains expected components
			for _, want := range tt.wantContains {
				if !strings.Contains(url, want) {
					t.Errorf("URL missing expected component %q\nGot URL: %s", want, url)
				}
			}

			// Verify it's a valid URL format
			if !strings.HasPrefix(url, "https://console.cloud.google.com/logs/query") {
				t.Errorf("URL should start with Cloud Console logs query endpoint")
			}

			// Verify query parameters are properly encoded
			if strings.Contains(url, " ") || strings.Contains(url, "\n") {
				t.Errorf("URL contains unencoded spaces or newlines: %s", url)
			}
		})
	}
}

func TestJSONFormattingConsistency(t *testing.T) {
	identity := "test-reconciler"

	status := Status[TestDetails]{
		ObservedGeneration: "test123",
		Status:             "in_progress",
		Details: TestDetails{
			Message: "Test",
			Count:   1,
		},
	}

	output, err := buildCheckRunOutput(identity, &status)
	if err != nil {
		t.Fatalf("buildCheckRunOutput() error = %v", err)
	}

	// Verify JSON is properly indented (2 spaces as per MarshalIndent)
	if !strings.Contains(output, "  \"observedGeneration\"") {
		t.Error("JSON should be indented with 2 spaces")
	}

	// Verify the JSON can be extracted and unmarshaled manually
	start := strings.Index(output, "<!--\n") + 5
	end := strings.Index(output, "\n-->")
	if start < 5 || end < 0 {
		t.Fatal("Could not find JSON boundaries")
	}

	jsonStr := output[start:end]
	var testStatus Status[TestDetails]
	if err := json.Unmarshal([]byte(jsonStr), &testStatus); err != nil {
		t.Errorf("Failed to manually unmarshal extracted JSON: %v", err)
	}

	// Verify it matches the original
	if diff := cmp.Diff(status, testStatus); diff != "" {
		t.Errorf("Manually extracted JSON mismatch (-want +got):\n%s", diff)
	}
}

func TestReadOnlyStatusManager(t *testing.T) {
	// Create a read-only status manager
	sm := &StatusManager[TestDetails]{
		identity:    "test-reconciler",
		projectID:   "test-project",
		serviceName: "test-service",
		readOnly:    true,
	}

	// Create a session
	res := &githubreconciler.Resource{
		Owner:  "test-owner",
		Repo:   "test-repo",
		Number: 123,
		Type:   githubreconciler.ResourceTypePullRequest,
		URL:    "https://github.com/test-owner/test-repo/pull/123",
	}

	session := sm.NewSession(nil, res, "abc123")

	// Verify session is read-only
	if !session.readOnly {
		t.Error("Session should be read-only")
	}

	// Attempt to set actual state should fail
	status := &Status[TestDetails]{
		Status: "completed",
		Details: TestDetails{
			Message: "Test",
		},
	}

	err := session.SetActualState(nil, "Test", status)
	if err == nil {
		t.Fatal("SetActualState should fail for read-only session")
	}

	if !strings.Contains(err.Error(), "read-only") {
		t.Errorf("Error should mention read-only, got: %v", err)
	}
}
