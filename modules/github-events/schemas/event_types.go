package schemas

import "time"

type Wrapper[T any] struct {
	When time.Time
	Body T
}

type User struct {
	Name string
}

type Organization struct {
	Name string
}

type Repo struct {
	Owner        User
	Organization Organization
	Name         string
}

type PullRequest struct {
	State          string
	Title          string
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Draft          bool
	Merged         bool
	MergeCommitSHA string `json:"merge_commit_sha"`
}

// https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request
// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestEvent
type PullRequestEvent struct {
	Action   string // assigned,opened  etc.
	Sender   User
	Assignee User
	Number   int
	Repo     Repo

	PullRequest PullRequest `json:"pull_request" bigquery:"pullRequest"`

	// Populated when action is synchronize
	Before string
	After  string
}

type Workflow struct {
	ID        int64 `bigquery:"id"`
	Name      string
	Path      string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	State     string
}

type WorkflowRun struct {
	ID         int64  `bigquery:"id"`
	RunNumber  int64  `bigquery:"runNumber"`
	RunAttempt int64  `bigquery:"runAttempt"`
	HeadBranch string `bigquery:"headBranch"`
	HeadSHA    string `bigquery:"headSHA"`

	Name       string
	Event      string
	Conclusion string // success, failure, cancelled, etc.
}

// https://docs.github.com/developers/webhooks-and-events/webhook-events-and-payloads#workflow_run
// subset of https://pkg.go.dev/github.com/google/go-github/v60/github#WorkflowRunEvent
type WorkflowRunEvent struct {
	Action      string // completed, etc.
	Workflow    Workflow
	WorkflowRun WorkflowRun
	Org         Organization
	Repo        Repo
	Sender      User
}
