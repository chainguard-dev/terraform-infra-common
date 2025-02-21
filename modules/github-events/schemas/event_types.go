package schemas

import (
	"time"

	"cloud.google.com/go/bigquery"
)

type Wrapper[T any] struct {
	When    time.Time
	Headers *GitHubHeaders
	Body    T
}

type GitHubHeaders struct {
	HookID                 bigquery.NullString `json:"hook_id,omitempty" bigquery:"hook_id"`
	DeliveryID             bigquery.NullString `json:"delivery_id,omitempty" bigquery:"delivery_id"`
	UserAgent              bigquery.NullString `json:"user_agent,omitempty" bigquery:"user_agent"`
	Event                  bigquery.NullString `json:"event,omitempty" bigquery:"event"`
	InstallationTargetType bigquery.NullString `json:"installation_target_type,omitempty" bigquery:"installation_target_type"`
	InstallationTargetID   bigquery.NullString `json:"installation_target_id,omitempty" bigquery:"installation_target_id"`
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
	Owner    User                `json:"owner,omitempty" bigquery:"owner"`
	Name     bigquery.NullString `json:"name,omitempty" bigquery:"name"`
	URL      bigquery.NullString `json:"url,omitempty" bigquery:"url"`
	FullName bigquery.NullString `json:"full_name,omitempty" bigquery:"full_name"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Installation
type Installation struct {
	// Installation ID
	ID bigquery.NullInt64 `json:"id,omitempty" bigquery:"id"`
	// App ID
	AppID bigquery.NullInt64 `json:"app_id,omitempty" bigquery:"app_id"`
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
	Action       bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	Sender       User                `json:"sender,omitempty" bigquery:"sender"`
	Assignee     User                `json:"assignee,omitempty" bigquery:"assignee"`
	Repository   Repository          `json:"repository,omitempty" bigquery:"repository"`
	Organization Organization        `json:"organization,omitempty" bigquery:"organization"`

	PullRequest PullRequest `json:"pull_request,omitempty" bigquery:"pull_request"`

	// Populated when action is synchronize
	Before bigquery.NullString `json:"before,omitempty" bigquery:"before"`
	After  bigquery.NullString `json:"after,omitempty" bigquery:"after"`

	Installation *Installation `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v68/github#PushEventRepository
type PushEventRepository struct {
	Owner    User                `json:"owner,omitempty" bigquery:"owner"`
	Name     bigquery.NullString `json:"name,omitempty" bigquery:"name"`
	URL      bigquery.NullString `json:"url,omitempty" bigquery:"url"`
	FullName bigquery.NullString `json:"full_name,omitempty" bigquery:"full_name"`
}

// https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#push
// https://pkg.go.dev/github.com/google/go-github/v68/github#PushEvent
type PushEvent struct {
	PushID       bigquery.NullInt64  `json:"push_id,omitempty" bigquery:"push_id"`
	Head         bigquery.NullString `json:"head,omitempty" bigquery:"head"`
	Ref          bigquery.NullString `json:"ref,omitempty" bigquery:"ref"`
	Size         bigquery.NullInt64  `json:"size,omitempty" bigquery:"size"`
	Before       bigquery.NullString `json:"before,omitempty" bigquery:"before"`
	DistinctSize bigquery.NullInt64  `json:"distinct_size,omitempty" bigquery:"distinct_size"`

	// The following fields are only populated by Webhook events.
	Action  bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	After   bigquery.NullString `json:"after,omitempty" bigquery:"after"`
	BaseRef bigquery.NullString `json:"base_ref,omitempty" bigquery:"base_ref"`
	Repo    PushEventRepository `json:"repository,omitempty" bigquery:"repository"`
	Sender  User                `json:"sender,omitempty" bigquery:"sender"`

	Organization Organization `json:"organization,omitempty" bigquery:"organization"`

	Installation *Installation `json:"installation,omitempty" bigquery:"installation"`
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
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#IssueCommentEvent
type IssueCommentEvent struct {
	Action       bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	Issue        Issue               `json:"issue,omitempty" bigquery:"issue"`
	Comment      IssueComment        `json:"comment,omitempty" bigquery:"comment"`
	Repo         Repository          `json:"repository,omitempty" bigquery:"repository"`
	Sender       User                `json:"sender,omitempty" bigquery:"sender"`
	Organization Organization        `json:"organization,omitempty" bigquery:"organization"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
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
	Installation      *Installation          `json:"installation,omitempty" bigquery:"installation"`
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

// https://pkg.go.dev/github.com/google/go-github/v60/github#CheckRunEvent
type CheckRunEvent struct {
	Action       bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	CheckRun     CheckRun            `json:"check_run,omitempty" bigquery:"check_run"`
	Repository   Repository          `json:"repository,omitempty" bigquery:"repository"`
	Organization Organization        `json:"organization,omitempty" bigquery:"organization"`
	Sender       User                `json:"sender,omitempty" bigquery:"sender"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#CheckRun
type CheckRun struct {
	ID           bigquery.NullInt64     `json:"id,omitempty" bigquery:"id"`
	HeadSHA      bigquery.NullString    `json:"head_sha,omitempty" bigquery:"head_sha"`
	Status       bigquery.NullString    `json:"status,omitempty" bigquery:"status"`
	Conclusion   bigquery.NullString    `json:"conclusion,omitempty" bigquery:"conclusion"`
	StartedAt    bigquery.NullTimestamp `json:"started_at,omitempty" bigquery:"started_at"`
	CompletedAt  bigquery.NullTimestamp `json:"completed_at,omitempty" bigquery:"completed_at"`
	Name         bigquery.NullString    `json:"name,omitempty" bigquery:"name"`
	CheckSuite   *CheckSuite            `json:"check_suite,omitempty" bigquery:"check_suite"`
	PullRequests []PullRequest          `json:"pull_requests,omitempty" bigquery:"pull_requests"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#CheckSuite
type CheckSuite struct {
	ID           bigquery.NullInt64     `json:"id,omitempty" bigquery:"id"`
	HeadSHA      bigquery.NullString    `json:"head_sha,omitempty" bigquery:"head_sha"`
	Status       bigquery.NullString    `json:"status,omitempty" bigquery:"status"`
	Conclusion   bigquery.NullString    `json:"conclusion,omitempty" bigquery:"conclusion"`
	CreatedAt    bigquery.NullTimestamp `json:"created_at,omitempty" bigquery:"created_at"`
	UpdatedAt    bigquery.NullTimestamp `json:"updated_at,omitempty" bigquery:"updated_at"`
	Repository   Repository             `json:"repository,omitempty" bigquery:"repository"`
	PullRequests []PullRequest          `json:"pull_requests,omitempty" bigquery:"pull_requests"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#CheckSuiteEvent
type CheckSuiteEvent struct {
	Action       bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	CheckSuite   *CheckSuite         `json:"check_suite,omitempty" bigquery:"check_suite"`
	Repository   Repository          `json:"repository,omitempty" bigquery:"repository"`
	Organization Organization        `json:"organization,omitempty" bigquery:"organization"`
	Sender       User                `json:"sender,omitempty" bigquery:"sender"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://github.com/google/go-github/blob/v60.0.0/github/event_types.go#L1085
type ProjectV2Item struct {
	ID            bigquery.NullInt64     `json:"id,omitempty" bigquery:"id"`
	NodeID        bigquery.NullString    `json:"node_id,omitempty" bigquery:"node_id"`
	ProjectNodeID bigquery.NullString    `json:"project_node_id,omitempty" bigquery:"project_node_id"`
	ContentNodeID bigquery.NullString    `json:"content_node_id,omitempty" bigquery:"content_node_id"`
	ContentType   bigquery.NullString    `json:"content_type,omitempty" bigquery:"content_type"`
	Creator       *User                  `json:"creator,omitempty" bigquery:"creator"`
	CreatedAt     bigquery.NullTimestamp `json:"created_at,omitempty" bigquery:"created_at"`
	UpdatedAt     bigquery.NullTimestamp `json:"updated_at,omitempty" bigquery:"updated_at"`
	ArchivedAt    bigquery.NullTimestamp `json:"archived_at,omitempty" bigquery:"archived_at"`
}

// https://github.com/google/go-github/blob/v60.0.0/github/event_types.go#L1062
type ProjectsV2ItemEvent struct {
	Action        bigquery.NullString `json:"action,omitempty" bigquery:"action"`
	Changes       bigquery.NullJSON   `json:"changes,omitempty" bigquery:"changes"`
	ProjectV2Item *ProjectV2Item      `json:"projects_v2_item,omitempty" bigquery:"projects_v2_item"`
	Organization  *Organization       `json:"organization,omitempty" bigquery:"organization"`
	Sender        *User               `json:"sender,omitempty" bigquery:"sender"`
	Installation  *Installation       `json:"installation,omitempty" bigquery:"installation"`
}
