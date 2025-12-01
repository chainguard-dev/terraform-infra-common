/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package issuemanager provides a reconciler-style abstraction for managing
// GitHub Issues based on desired state. It discovers current state, compares
// it to desired state, and performs the necessary create/update/close operations
// to align reality with intent.
//
// # Reconciliation Model
//
// IssueManager follows a declarative reconciliation pattern:
//
//  1. Discover Current State: Query GitHub for existing issues using labels
//     to identify issues managed by this reconciler instance
//
//  2. Extract Embedded Data: Read structured data embedded in issue bodies
//     to understand the current state each issue represents
//
//  3. Compare with Desired State: Match existing issues to desired states
//     using the Equal method defined on the data type
//
//  4. Reconcile Differences:
//     - Create new issues for desired states with no existing issue
//     - Update existing issues if their embedded data differs from desired
//     - Close existing issues that no longer have a corresponding desired state
//
// This makes it ideal for reconcilers that need to maintain a set of GitHub
// issues reflecting some external state (e.g., scan results, configuration
// drift, policy violations).
//
// # Session-Based Discovery
//
// Each reconciliation begins with a Session that discovers current state:
//
//   - NewSession queries GitHub for all open issues matching the identity label
//   - Each issue's body is parsed to extract embedded structured data
//   - The Session maintains a map of existing issues keyed by their data
//   - Issues with skip labels (skip:{identity}) are preserved during reconciliation
//
// The Session provides a snapshot of current state for the reconciliation cycle.
//
// # Desired State Reconciliation
//
// The reconciler declares desired state as a slice of data objects:
//
//   - Reconcile accepts the desired states and performs a complete reconciliation
//   - Each data object's Equal method determines if an existing issue corresponds to it
//   - For matches, issues are updated only if the embedded data changed
//   - For non-matches, new issues are created
//   - Issues not in the desired set are automatically closed
//   - Issues with skip labels are preserved throughout all phases
//
// This ensures GitHub issues always reflect the latest desired state.
//
// # Generic Type Parameter
//
// IssueManager is generic over type T, which must implement the Comparable[T]
// interface. This interface requires an Equal(T) bool method that determines
// when two instances represent the same logical issue.
//
// The type T represents the structured data embedded in each issue. This data:
//
//   - Populates Go templates for issue titles and bodies
//   - Is embedded as JSON in HTML comments within the issue body
//   - Determines whether an issue needs updating (by comparing embedded vs desired)
//   - Enables matching between existing and desired issues via the Equal method
//
// The type T must be JSON-serializable and contain the fields needed to
// identify and describe the issue's purpose. The Equal method typically
// compares identifying fields (like IDs) rather than all fields.
//
// # Identity Length Limit
//
// The identity parameter must not exceed 20 characters (maxIdentityLength).
// This ensures that labels constructed as "identity:path" stay within GitHub's
// 50 character label limit (maxGitHubLabelLength). When the combined length
// exceeds 50, the path is replaced with a truncated SHA256 hash.
//
// # Usage Example
//
// Create an IssueManager with templates for title and body:
//
//	type IssueData struct {
//	    ID       string
//	    Status   string
//	    Priority string
//	}
//
//	// Equal implements the Comparable interface for IssueData.
//	// Issues are considered the same if they have the same ID.
//	func (d IssueData) Equal(other IssueData) bool {
//	    return d.ID == other.ID
//	}
//
//	titleTmpl := template.Must(template.New("title").Parse("Issue {{.ID}}"))
//	bodyTmpl := template.Must(template.New("body").Parse("Status: {{.Status}}\nPriority: {{.Priority}}"))
//
//	im := issuemanager.New[IssueData]("my-reconciler", titleTmpl, bodyTmpl)
//
// Optionally add label templates to generate dynamic labels from data:
//
//	labelTmpl := template.Must(template.New("priority").Parse("priority:{{.Priority}}"))
//	im := issuemanager.New[IssueData]("my-reconciler", titleTmpl, bodyTmpl,
//	    issuemanager.WithLabelTemplates(labelTmpl),
//	    issuemanager.WithMaxDesiredIssuesPerPath(1), // default, increase with caution
//	)
//
// Start a reconciliation session to discover current state:
//
//	session, err := im.NewSession(ctx, ghClient, "owner/repo")
//	if err != nil {
//	    return err
//	}
//
// Reconcile to desired state with a single call.
// This performs create, update, and close operations atomically.
// Issues with skip labels are automatically preserved:
//
//	desiredStates := []*IssueData{
//	    {ID: "001", Status: "active", Priority: "high"},
//	    {ID: "002", Status: "pending", Priority: "medium"},
//	}
//
//	urls, err := session.Reconcile(ctx, desiredStates, []string{"automated"}, "No longer relevant")
//	if err != nil {
//	    return err
//	}
//
// This ensures exactly the desired set of issues exists and is up-to-date.
package issuemanager
