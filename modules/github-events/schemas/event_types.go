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
	Login string
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Organization
type Organization struct {
	Login string
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Repository
type Repo struct {
	Owner        User
	Organization Organization
	Name         string
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequest
type PullRequest struct {
	Number int64
	State  string
	Title  string

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
	Action   string // assigned,opened  etc.
	Sender   User
	Assignee User
	Repo     Repo

	PullRequest PullRequest

	// Populated when action is synchronize
	Before string
	After  string
}

type Workflow struct {
	ID    bigquery.NullInt64 `json:"id,omitempty" bigquery:"id"`
	Name  string
	Path  string
	State string
	// CreatedAt *Timestamp `json:"created_at,omitempty"`
	// UpdatedAt *Timestamp `json:"updated_at,omitempty"`
}

type WorkflowRun struct {
	ID         int64 `bigquery:"id"`
	RunNumber  int64
	RunAttempt int64
	HeadBranch string
	HeadSHA    string

	Name       string
	Event      string
	Conclusion string // success, failure, cancelled, etc.
}

// https://docs.github.com/developers/webhooks-and-events/webhook-events-and-payloads#workflow_run
// subset of https://pkg.go.dev/github.com/google/go-github/v60/github#WorkflowRunEvent
type WorkflowRunEvent struct {
	Action      string // completed, etc.
	Workflow    Workflow
	WorkflowRun WorkflowRun `json:"workflow_run"`
	Org         Organization
	Repo        Repo
	Sender      User
}
