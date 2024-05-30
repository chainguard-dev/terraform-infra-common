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

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestBranch
type PullRequestBranch struct {
	Ref  bigquery.NullString `json:"ref,omitempty" bigquery:"ref"`
	SHA  bigquery.NullString `json:"sha,omitempty" bigquery:"sha"`
	Repo Repository          `json:"repo,omitempty" bigquery:"repo"`
	User User                `json:"user,omitempty" bigquery:"user"`
}

type Label struct {
	Name bigquery.NullString `json:"name,omitempty" bigquery:"name"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequest
type PullRequest struct {
	Number bigquery.NullInt64  `json:"number,omitempty" bigquery:"number"`
	State  bigquery.NullString `json:"state,omitempty" bigquery:"state"`
	Title  bigquery.NullString `json:"title,omitempty" bigquery:"title"`

	Base PullRequestBranch `json:"base,omitempty" bigquery:"base"`
	Head PullRequestBranch `json:"head,omitempty" bigquery:"head"`

	Labels []Label `json:"labels" bigquery:"labels"`

	CreatedAt bigquery.NullTimestamp `json:"created_at,omitempty" bigquery:"created_at"`
	UpdatedAt bigquery.NullTimestamp `json:"updated_at,omitempty" bigquery:"updated_at"`
	ClosedAt  bigquery.NullTimestamp `json:"closed_at,omitempty" bigquery:"closed_at"`
	MergedAt  bigquery.NullTimestamp `json:"merged_at,omitempty" bigquery:"merged_at"`

	Mergeable      bigquery.NullBool   `json:"mergeable,omitempty" bigquery:"mergeable"`
	MergeableState bigquery.NullString `json:"mergeable_state,omitempty" bigquery:"mergeable_state"`
	MergedBy       User                `json:"merged_by,omitempty" bigquery:"merged_by"`
	MergeCommitSHA bigquery.NullString `json:"merge_commit_sha,omitempty" bigquery:"merge_commit_sha"`

	Additions    bigquery.NullInt64 `json:"additions,omitempty" bigquery:"additions"`
	Deletions    bigquery.NullInt64 `json:"deletions,omitempty" bigquery:"deletions"`
	ChangedFiles bigquery.NullInt64 `json:"changed_files,omitempty" bigquery:"changed_files"`
}

type PullRequestLinks struct {
	URL      bigquery.NullString    `json:"url,omitempty" bigquery:"url"`
	HTMLURL  bigquery.NullString    `json:"html_url,omitempty" bigquery:"html_url"`
	DiffURL  bigquery.NullString    `json:"diff_url,omitempty" bigquery:"diff_url"`
	PatchURL bigquery.NullString    `json:"patch_url,omitempty" bigquery:"patch_url"`
	MergedAt bigquery.NullTimestamp `json:"merged_at,omitempty" bigquery:"merged_at"`
}

// https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request
// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestEvent
type PullRequestEvent struct {
	// assigned,opened  etc.
	Action     bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	Sender     User                `json:"sender,omitempty" bigquery:"sender"`
	Assignee   User                `json:"assignee,omitempty" bigquery:"assignee"`
	Repository Repository          `json:"repository,omitempty" bigquery:"repository"`

	PullRequest PullRequest `json:"pull_request,omitempty" bigquery:"pull_request"`

	// Populated when action is synchronize
	Before bigquery.NullString `json:"before,omitempty" bigquery:"before"`
	After  bigquery.NullString `json:"after,omitempty" bigquery:"after"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Workflow
type Workflow struct {
	ID    bigquery.NullInt64  `json:"id,omitempty" bigquery:"id"`
	Name  bigquery.NullString `json:"name,omitempty" bigquery:"name"`
	Path  bigquery.NullString `json:"path,omitempty" bigquery:"path"`
	State bigquery.NullString `json:"state,omitempty" bigquery:"state"`

	CreatedAt bigquery.NullTimestamp `json:"created_at,omitempty" bigquery:"created_at"`
	UpdatedAt bigquery.NullTimestamp `json:"updated_at,omitempty" bigquery:"updated_at"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#WorkflowRun
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

// https://pkg.go.dev/github.com/google/go-github/v60/github#IssueCommentEvent
type IssueCommentEvent struct {
	Action       bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	Issue        Issue               `json:"issue,omitempty" bigquery:"issue"`
	Comment      IssueComment        `json:"comment,omitempty" bigquery:"comment"`
	Repo         Repository          `json:"repository,omitempty" bigquery:"repository"`
	Sender       User                `json:"sender,omitempty" bigquery:"sender"`
	Organization Organization        `json:"organization,omitempty" bigquery:"organization"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#IssueEvent
type IssueEvent struct {
	ID                bigquery.NullInt64     `json:"id,omitempty" bigquery:"id"`
	URL               bigquery.NullString    `json:"url,omitempty" bigquery:"url"`
	Actor             User                   `json:"actor,omitempty" bigquery:"actor"`
	Action            bigquery.NullString    `json:"action,omitempty" bigquery:"action"`
	Event             bigquery.NullString    `json:"event,omitempty" bigquery:"event"`
	CreatedAt         bigquery.NullTimestamp `json:"created_at,omitempty" bigquery:"created_at"`
	Issue             Issue                  `json:"issue,omitempty" bigquery:"issue"`
	Repository        Repository             `json:"repository,omitempty" bigquery:"repository"`
	Assignee          User                   `json:"assignee,omitempty" bigquery:"assignee"`
	Assigner          User                   `json:"assigner,omitempty" bigquery:"assigner"`
	CommitID          bigquery.NullString    `json:"commit_id,omitempty" bigquery:"commit_id"`
	Label             Label                  `json:"label,omitempty" bigquery:"label"`
	LockReason        bigquery.NullString    `json:"lock_reason,omitempty" bigquery:"lock_reason"`
	RequestedReviewer User                   `json:"requested_reviewer,omitempty" bigquery:"requested_reviewer"`
	ReviewRequester   User                   `json:"review_requester,omitempty" bigquery:"review_requester"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Issue
type Issue struct {
	ID                bigquery.NullInt64     `json:"id,omitempty" bigquery:"id"`
	Number            bigquery.NullInt64     `json:"number,omitempty" bigquery:"number"`
	State             bigquery.NullString    `json:"state,omitempty" bigquery:"state"`
	StateReason       bigquery.NullString    `json:"state_reason,omitempty" bigquery:"state_reason"`
	Locked            bigquery.NullBool      `json:"locked,omitempty" bigquery:"locked"`
	Title             bigquery.NullString    `json:"title,omitempty" bigquery:"title"`
	Body              bigquery.NullString    `json:"body,omitempty" bigquery:"body"`
	AuthorAssociation bigquery.NullString    `json:"author_association,omitempty" bigquery:"author_association"`
	User              User                   `json:"user,omitempty" bigquery:"user"`
	Labels            []Label                `json:"labels" bigquery:"labels"`
	Assignee          User                   `json:"assignee,omitempty" bigquery:"assignee"`
	Comments          bigquery.NullInt64     `json:"comments,omitempty" bigquery:"comments"`
	ClosedAt          bigquery.NullTimestamp `json:"closed_at,omitempty" bigquery:"closed_at"`
	CreatedAt         bigquery.NullTimestamp `json:"created_at,omitempty" bigquery:"created_at"`
	UpdatedAt         bigquery.NullTimestamp `json:"updated_at,omitempty" bigquery:"updated_at"`
	ClosedBy          User                   `json:"closed_by,omitempty" bigquery:"closed_by"`
	URL               bigquery.NullString    `json:"url,omitempty" bigquery:"url"`
	HTMLURL           bigquery.NullString    `json:"html_url,omitempty" bigquery:"html_url"`
	CommentsURL       bigquery.NullString    `json:"comments_url,omitempty" bigquery:"comments_url"`
	EventsURL         bigquery.NullString    `json:"events_url,omitempty" bigquery:"events_url"`
	LabelsURL         bigquery.NullString    `json:"labels_url,omitempty" bigquery:"labels_url"`
	RepositoryURL     bigquery.NullString    `json:"repository_url,omitempty" bigquery:"repository_url"`
	PullRequestLinks  PullRequestLinks       `json:"pull_request,omitempty" bigquery:"pull_request"`
	Repository        Repository             `json:"repository,omitempty" bigquery:"repository"`
	Assignees         []User                 `json:"assignees,omitempty" bigquery:"assignees"`
	NodeID            bigquery.NullString    `json:"node_id,omitempty" bigquery:"node_id"`
	Draft             bigquery.NullBool      `json:"draft,omitempty" bigquery:"draft"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#IssueComment
type IssueComment struct {
	URL      bigquery.NullString    `json:"url,omitempty" bigquery:"url"`
	HTMLURL  bigquery.NullString    `json:"html_url,omitempty" bigquery:"html_url"`
	DiffURL  bigquery.NullString    `json:"diff_url,omitempty" bigquery:"diff_url"`
	PatchURL bigquery.NullString    `json:"patch_url,omitempty" bigquery:"patch_url"`
	MergedAt bigquery.NullTimestamp `json:"merged_at,omitempty" bigquery:"merged_at"`
}
