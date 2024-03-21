package schemas

import (
	"time"

	"cloud.google.com/go/bigquery"
)

type Wrapper[T any] struct {
	When time.Time
	Body T
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#User
type User struct {
	Login bigquery.NullString `json:"login,omitempty" bigquery:"login"`
	Type  bigquery.NullString `json:"type,omitempty" bigquery:"type"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Organization
type Organization struct {
	Login bigquery.NullString `json:"login,omitempty" bigquery:"login"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Repository
type Repository struct {
	Owner    User
	Name     bigquery.NullString `json:"name,omitempty" bigquery:"name"`
	URL      bigquery.NullString `json:"url,omitempty" bigquery:"url"`
	FullName bigquery.NullString `json:"full_name,omitempty" bigquery:"full_name"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequest
type PullRequest struct {
	Number bigquery.NullInt64
	State  bigquery.NullString
	Title  bigquery.NullString

	// CreatedAt *Timestamp `json:"created_at,omitempty"`
	// UpdatedAt *Timestamp `json:"updated_at,omitempty"`
	// ClosedAt  *Timestamp `json:"closed_at,omitempty"`
	// MergedAt  *Timestamp `json:"merged_at,omitempty"`

	Mergeable      bigquery.NullBool
	MergeableState bigquery.NullString
	MergedBy       *User
	MergeCommitSHA bigquery.NullString

	Additions    bigquery.NullInt64
	Deletions    bigquery.NullInt64
	ChangedFiles bigquery.NullInt64
}

// https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request
// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestEvent
type PullRequestEvent struct {
	Action     bigquery.NullString // assigned,opened  etc.
	Sender     User
	Assignee   User
	Repository Repository

	PullRequest PullRequest

	// Populated when action is synchronize
	Before bigquery.NullString
	After  bigquery.NullString
}

type Workflow struct {
	ID    bigquery.NullInt64  `json:"id,omitempty" bigquery:"id"`
	Name  bigquery.NullString `json:"name,omitempty" bigquery:"name"`
	Path  bigquery.NullString `json:"path,omitempty" bigquery:"path"`
	State bigquery.NullString `json:"state,omitempty" bigquery:"state"`

	CreatedAt bigquery.NullTimestamp `json:"created_at,omitempty" bigquery:"created_at"`
	UpdatedAt bigquery.NullTimestamp `json:"updated_at,omitempty" bigquery:"updated_at"`
}

type WorkflowRun struct {
	ID           bigquery.NullInt64     `json:"id,omitempty" bigquery:"id"`
	RunNumber    bigquery.NullInt64     `json:"run_number,omitempty" bigquery:"run_number"`
	RunAttempt   bigquery.NullInt64     `json:"run_attempt,omitempty" bigquery:"run_attempt"`
	HeadBranch   bigquery.NullString    `json:"head_branch,omitempty" bigquery:"head_branch"`
	HeadSHA      bigquery.NullString    `json:"head_sha,omitempty" bigquery:"head_sha"`
	Name         bigquery.NullString    `json:"name,omitempty" bigquery:"name"`
	Event        bigquery.NullString    `json:"event,omitempty" bigquery:"event"`
	Status       bigquery.NullString    `json:"status,omitempty" bigquery:"status"`
	RunStartedAt bigquery.NullTimestamp `json:"run_started_at,omitempty" bigquery:"run_started_at"`

	// success, failure, cancelled, etc.
	Conclusion bigquery.NullString `json:"conclusion,omitempty" bigquery:"conclusion"`
}

// https://docs.github.com/developers/webhooks-and-events/webhook-events-and-payloads#workflow_run
// subset of https://pkg.go.dev/github.com/google/go-github/v60/github#WorkflowRunEvent
type WorkflowRunEvent struct {
	// completed, etc.
	Action       bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	Workflow     Workflow            `json:"workflow,omitempty" bigquery:"workflow"`
	WorkflowRun  WorkflowRun         `json:"workflow_run,omitempty" bigquery:"workflow_run"`
	Organization Organization        `json:"organization,omitempty" bigquery:"organization"`
	Repository   Repository          `json:"repository,omitempty" bigquery:"repository"`
	Sender       User                `json:"sender,omitempty" bigquery:"sender"`
}
