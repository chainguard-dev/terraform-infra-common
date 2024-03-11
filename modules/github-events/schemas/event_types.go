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

type PullRequestEvent struct {
	Action string // assigned,opened  etc.
	User   User
	Number int
	Repo   Repo

	// Populated when action is synchronize
	Before string
	After  string
}

type Workflow struct {
	ID        int64 `bigquery:"id"`
	Name      string
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
	State     string
}

type WorkflowRun struct {
	ID         int64 `bigquery:"id"`
	RunNumber  int64
	RunAttempt int64
	HeadBranch string
	HeadSHA    string `bigquery:"headSHA"`

	Name       string
	Event      string
	Conclusion string // success, failure, cancelled, etc.
}

// https://docs.github.com/developers/webhooks-and-events/webhook-events-and-payloads#workflow_run
type WorkflowRunEvent struct {
	Action       string // completed, etc.
	Workflow     Workflow
	Organization Organization
	Repo         Repo
	User         User
}
