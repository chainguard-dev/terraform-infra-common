/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

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
	HookID                 bigquery.NullString `json:"hook_id" bigquery:"hook_id"`
	DeliveryID             bigquery.NullString `json:"delivery_id" bigquery:"delivery_id"`
	UserAgent              bigquery.NullString `json:"user_agent" bigquery:"user_agent"`
	Event                  bigquery.NullString `json:"event" bigquery:"event"`
	InstallationTargetType bigquery.NullString `json:"installation_target_type" bigquery:"installation_target_type"`
	InstallationTargetID   bigquery.NullString `json:"installation_target_id" bigquery:"installation_target_id"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#User
type User struct {
	Login bigquery.NullString `json:"login" bigquery:"login"`
	Type  bigquery.NullString `json:"type" bigquery:"type"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Organization
type Organization struct {
	Login bigquery.NullString `json:"login" bigquery:"login"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Repository
type Repository struct {
	Owner    User                `json:"owner" bigquery:"owner"`
	Name     bigquery.NullString `json:"name" bigquery:"name"`
	URL      bigquery.NullString `json:"url" bigquery:"url"`
	FullName bigquery.NullString `json:"full_name" bigquery:"full_name"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Installation
type Installation struct {
	// Installation ID
	ID bigquery.NullInt64 `json:"id" bigquery:"id"`
	// App ID
	AppID bigquery.NullInt64 `json:"app_id" bigquery:"app_id"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestBranch
type PullRequestBranch struct {
	Ref  bigquery.NullString `json:"ref" bigquery:"ref"`
	SHA  bigquery.NullString `json:"sha" bigquery:"sha"`
	Repo Repository          `json:"repo" bigquery:"repo"`
	User User                `json:"user" bigquery:"user"`
}

type Label struct {
	Name bigquery.NullString `json:"name" bigquery:"name"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequest
type PullRequest struct {
	Number bigquery.NullInt64  `json:"number" bigquery:"number"`
	State  bigquery.NullString `json:"state" bigquery:"state"`
	Title  bigquery.NullString `json:"title" bigquery:"title"`

	Base PullRequestBranch `json:"base" bigquery:"base"`
	Head PullRequestBranch `json:"head" bigquery:"head"`

	Labels []Label `json:"labels" bigquery:"labels"`

	CreatedAt bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	UpdatedAt bigquery.NullTimestamp `json:"updated_at" bigquery:"updated_at"`
	ClosedAt  bigquery.NullTimestamp `json:"closed_at" bigquery:"closed_at"`
	MergedAt  bigquery.NullTimestamp `json:"merged_at" bigquery:"merged_at"`

	Mergeable      bigquery.NullBool   `json:"mergeable" bigquery:"mergeable"`
	MergeableState bigquery.NullString `json:"mergeable_state" bigquery:"mergeable_state"`
	MergedBy       User                `json:"merged_by" bigquery:"merged_by"`
	MergeCommitSHA bigquery.NullString `json:"merge_commit_sha" bigquery:"merge_commit_sha"`

	Additions    bigquery.NullInt64 `json:"additions" bigquery:"additions"`
	Deletions    bigquery.NullInt64 `json:"deletions" bigquery:"deletions"`
	ChangedFiles bigquery.NullInt64 `json:"changed_files" bigquery:"changed_files"`
}

type PullRequestLinks struct {
	URL      bigquery.NullString    `json:"url" bigquery:"url"`
	HTMLURL  bigquery.NullString    `json:"html_url" bigquery:"html_url"`
	DiffURL  bigquery.NullString    `json:"diff_url" bigquery:"diff_url"`
	PatchURL bigquery.NullString    `json:"patch_url" bigquery:"patch_url"`
	MergedAt bigquery.NullTimestamp `json:"merged_at" bigquery:"merged_at"`
}

// https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request
// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestEvent
type PullRequestEvent struct {
	// assigned,opened  etc.
	Action       bigquery.NullString `json:"action" bigquery:"action"`
	Sender       User                `json:"sender" bigquery:"sender"`
	Assignee     User                `json:"assignee" bigquery:"assignee"`
	Repository   Repository          `json:"repository" bigquery:"repository"`
	Organization Organization        `json:"organization" bigquery:"organization"`

	PullRequest PullRequest `json:"pull_request" bigquery:"pull_request"`

	// Populated when action is synchronize
	Before bigquery.NullString `json:"before" bigquery:"before"`
	After  bigquery.NullString `json:"after" bigquery:"after"`

	Installation *Installation `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v68/github#PushEventRepository
type PushEventRepository struct {
	Owner    User                `json:"owner" bigquery:"owner"`
	Name     bigquery.NullString `json:"name" bigquery:"name"`
	URL      bigquery.NullString `json:"url" bigquery:"url"`
	FullName bigquery.NullString `json:"full_name" bigquery:"full_name"`
}

// https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#push
// https://pkg.go.dev/github.com/google/go-github/v68/github#PushEvent
type PushEvent struct {
	PushID       bigquery.NullInt64  `json:"push_id" bigquery:"push_id"`
	Head         bigquery.NullString `json:"head" bigquery:"head"`
	Ref          bigquery.NullString `json:"ref" bigquery:"ref"`
	Size         bigquery.NullInt64  `json:"size" bigquery:"size"`
	Before       bigquery.NullString `json:"before" bigquery:"before"`
	DistinctSize bigquery.NullInt64  `json:"distinct_size" bigquery:"distinct_size"`

	// The following fields are only populated by Webhook events.
	Action  bigquery.NullString `json:"action" bigquery:"action"`
	After   bigquery.NullString `json:"after" bigquery:"after"`
	BaseRef bigquery.NullString `json:"base_ref" bigquery:"base_ref"`
	Forced  bigquery.NullBool   `json:"forced" bigquery:"forced"`
	Repo    PushEventRepository `json:"repository" bigquery:"repository"`
	Sender  User                `json:"sender" bigquery:"sender"`

	Organization Organization `json:"organization" bigquery:"organization"`

	Installation *Installation `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Workflow
type Workflow struct {
	ID    bigquery.NullInt64  `json:"id" bigquery:"id"`
	Name  bigquery.NullString `json:"name" bigquery:"name"`
	Path  bigquery.NullString `json:"path" bigquery:"path"`
	State bigquery.NullString `json:"state" bigquery:"state"`

	CreatedAt bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	UpdatedAt bigquery.NullTimestamp `json:"updated_at" bigquery:"updated_at"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#WorkflowRun
//
// Note: GitHub does not include `completed_at` on the workflow_run object in
// webhook payloads. For action="completed" events, `updated_at` is the moment
// the run reached terminal state. For accurate per-job/per-step duration metrics
// use the workflow_job event instead.
type WorkflowRun struct {
	ID           bigquery.NullInt64     `json:"id" bigquery:"id"`
	RunNumber    bigquery.NullInt64     `json:"run_number" bigquery:"run_number"`
	RunAttempt   bigquery.NullInt64     `json:"run_attempt" bigquery:"run_attempt"`
	HeadBranch   bigquery.NullString    `json:"head_branch" bigquery:"head_branch"`
	HeadSHA      bigquery.NullString    `json:"head_sha" bigquery:"head_sha"`
	Name         bigquery.NullString    `json:"name" bigquery:"name"`
	Event        bigquery.NullString    `json:"event" bigquery:"event"`
	Status       bigquery.NullString    `json:"status" bigquery:"status"`
	CreatedAt    bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	UpdatedAt    bigquery.NullTimestamp `json:"updated_at" bigquery:"updated_at"`
	RunStartedAt bigquery.NullTimestamp `json:"run_started_at" bigquery:"run_started_at"`
	// CompletedAt is retained as a noop field. GitHub does not populate it on
	// workflow_run payloads (see struct comment above), so it is always NULL.
	// Removing it would change the BQ schema and trigger table recreation,
	// which is blocked by deletion_protection on the recorder. Use
	// updated_at on action="completed" events, or workflow_job.completed_at,
	// for the run's terminal-state timestamp.
	CompletedAt bigquery.NullTimestamp `json:"completed_at" bigquery:"completed_at"`

	// success, failure, cancelled, etc.
	Conclusion bigquery.NullString `json:"conclusion" bigquery:"conclusion"`
}

// https://docs.github.com/developers/webhooks-and-events/webhook-events-and-payloads#workflow_run
// subset of https://pkg.go.dev/github.com/google/go-github/v60/github#WorkflowRunEvent
type WorkflowRunEvent struct {
	// completed, etc.
	Action       bigquery.NullString `json:"action" bigquery:"action"`
	Workflow     Workflow            `json:"workflow" bigquery:"workflow"`
	WorkflowRun  WorkflowRun         `json:"workflow_run" bigquery:"workflow_run"`
	Organization Organization        `json:"organization" bigquery:"organization"`
	Repository   Repository          `json:"repository" bigquery:"repository"`
	Sender       User                `json:"sender" bigquery:"sender"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#TaskStep
type WorkflowJobStep struct {
	Name        bigquery.NullString    `json:"name" bigquery:"name"`
	Status      bigquery.NullString    `json:"status" bigquery:"status"`
	Conclusion  bigquery.NullString    `json:"conclusion" bigquery:"conclusion"`
	Number      bigquery.NullInt64     `json:"number" bigquery:"number"`
	StartedAt   bigquery.NullTimestamp `json:"started_at" bigquery:"started_at"`
	CompletedAt bigquery.NullTimestamp `json:"completed_at" bigquery:"completed_at"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#WorkflowJob
type WorkflowJob struct {
	ID              bigquery.NullInt64     `json:"id" bigquery:"id"`
	RunID           bigquery.NullInt64     `json:"run_id" bigquery:"run_id"`
	RunAttempt      bigquery.NullInt64     `json:"run_attempt" bigquery:"run_attempt"`
	WorkflowName    bigquery.NullString    `json:"workflow_name" bigquery:"workflow_name"`
	HeadBranch      bigquery.NullString    `json:"head_branch" bigquery:"head_branch"`
	HeadSHA         bigquery.NullString    `json:"head_sha" bigquery:"head_sha"`
	Name            bigquery.NullString    `json:"name" bigquery:"name"`
	Status          bigquery.NullString    `json:"status" bigquery:"status"`
	Conclusion      bigquery.NullString    `json:"conclusion" bigquery:"conclusion"`
	CreatedAt       bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	StartedAt       bigquery.NullTimestamp `json:"started_at" bigquery:"started_at"`
	CompletedAt     bigquery.NullTimestamp `json:"completed_at" bigquery:"completed_at"`
	RunnerID        bigquery.NullInt64     `json:"runner_id" bigquery:"runner_id"`
	RunnerName      bigquery.NullString    `json:"runner_name" bigquery:"runner_name"`
	RunnerGroupID   bigquery.NullInt64     `json:"runner_group_id" bigquery:"runner_group_id"`
	RunnerGroupName bigquery.NullString    `json:"runner_group_name" bigquery:"runner_group_name"`
	Labels          []string               `json:"labels,omitempty" bigquery:"labels"`
	Steps           []WorkflowJobStep      `json:"steps,omitempty" bigquery:"steps"`
}

// https://docs.github.com/en/webhooks/webhook-events-and-payloads#workflow_job
// subset of https://pkg.go.dev/github.com/google/go-github/v60/github#WorkflowJobEvent
type WorkflowJobEvent struct {
	// queued, in_progress, completed, waiting
	Action       bigquery.NullString `json:"action" bigquery:"action"`
	WorkflowJob  WorkflowJob         `json:"workflow_job" bigquery:"workflow_job"`
	Organization Organization        `json:"organization" bigquery:"organization"`
	Repository   Repository          `json:"repository" bigquery:"repository"`
	Sender       User                `json:"sender" bigquery:"sender"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#IssueCommentEvent
type IssueCommentEvent struct {
	Action       bigquery.NullString `json:"action" bigquery:"action"`
	Issue        Issue               `json:"issue" bigquery:"issue"`
	Comment      IssueComment        `json:"comment" bigquery:"comment"`
	Repo         Repository          `json:"repository" bigquery:"repository"`
	Sender       User                `json:"sender" bigquery:"sender"`
	Organization Organization        `json:"organization" bigquery:"organization"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#IssueEvent
type IssueEvent struct {
	ID                bigquery.NullInt64     `json:"id" bigquery:"id"`
	URL               bigquery.NullString    `json:"url" bigquery:"url"`
	Actor             User                   `json:"actor" bigquery:"actor"`
	Action            bigquery.NullString    `json:"action" bigquery:"action"`
	Event             bigquery.NullString    `json:"event" bigquery:"event"`
	CreatedAt         bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	Issue             Issue                  `json:"issue" bigquery:"issue"`
	Repository        Repository             `json:"repository" bigquery:"repository"`
	Assignee          User                   `json:"assignee" bigquery:"assignee"`
	Assigner          User                   `json:"assigner" bigquery:"assigner"`
	CommitID          bigquery.NullString    `json:"commit_id" bigquery:"commit_id"`
	Label             Label                  `json:"label" bigquery:"label"`
	LockReason        bigquery.NullString    `json:"lock_reason" bigquery:"lock_reason"`
	RequestedReviewer User                   `json:"requested_reviewer" bigquery:"requested_reviewer"`
	ReviewRequester   User                   `json:"review_requester" bigquery:"review_requester"`
	Installation      *Installation          `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#Issue
type Issue struct {
	ID                bigquery.NullInt64     `json:"id" bigquery:"id"`
	Number            bigquery.NullInt64     `json:"number" bigquery:"number"`
	State             bigquery.NullString    `json:"state" bigquery:"state"`
	StateReason       bigquery.NullString    `json:"state_reason" bigquery:"state_reason"`
	Locked            bigquery.NullBool      `json:"locked" bigquery:"locked"`
	Title             bigquery.NullString    `json:"title" bigquery:"title"`
	Body              bigquery.NullString    `json:"body" bigquery:"body"`
	AuthorAssociation bigquery.NullString    `json:"author_association" bigquery:"author_association"`
	User              User                   `json:"user" bigquery:"user"`
	Labels            []Label                `json:"labels" bigquery:"labels"`
	Assignee          User                   `json:"assignee" bigquery:"assignee"`
	Comments          bigquery.NullInt64     `json:"comments" bigquery:"comments"`
	ClosedAt          bigquery.NullTimestamp `json:"closed_at" bigquery:"closed_at"`
	CreatedAt         bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	UpdatedAt         bigquery.NullTimestamp `json:"updated_at" bigquery:"updated_at"`
	ClosedBy          User                   `json:"closed_by" bigquery:"closed_by"`
	URL               bigquery.NullString    `json:"url" bigquery:"url"`
	HTMLURL           bigquery.NullString    `json:"html_url" bigquery:"html_url"`
	CommentsURL       bigquery.NullString    `json:"comments_url" bigquery:"comments_url"`
	EventsURL         bigquery.NullString    `json:"events_url" bigquery:"events_url"`
	LabelsURL         bigquery.NullString    `json:"labels_url" bigquery:"labels_url"`
	RepositoryURL     bigquery.NullString    `json:"repository_url" bigquery:"repository_url"`
	PullRequestLinks  PullRequestLinks       `json:"pull_request" bigquery:"pull_request"`
	Repository        Repository             `json:"repository" bigquery:"repository"`
	Assignees         []User                 `json:"assignees,omitempty" bigquery:"assignees"`
	NodeID            bigquery.NullString    `json:"node_id" bigquery:"node_id"`
	Draft             bigquery.NullBool      `json:"draft" bigquery:"draft"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#IssueComment
type IssueComment struct {
	URL      bigquery.NullString    `json:"url" bigquery:"url"`
	HTMLURL  bigquery.NullString    `json:"html_url" bigquery:"html_url"`
	DiffURL  bigquery.NullString    `json:"diff_url" bigquery:"diff_url"`
	PatchURL bigquery.NullString    `json:"patch_url" bigquery:"patch_url"`
	MergedAt bigquery.NullTimestamp `json:"merged_at" bigquery:"merged_at"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#CheckRunEvent
type CheckRunEvent struct {
	Action       bigquery.NullString `json:"action" bigquery:"action"`
	CheckRun     CheckRun            `json:"check_run" bigquery:"check_run"`
	Repository   Repository          `json:"repository" bigquery:"repository"`
	Organization Organization        `json:"organization" bigquery:"organization"`
	Sender       User                `json:"sender" bigquery:"sender"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#CheckRun
type CheckRun struct {
	ID           bigquery.NullInt64     `json:"id" bigquery:"id"`
	HeadSHA      bigquery.NullString    `json:"head_sha" bigquery:"head_sha"`
	Status       bigquery.NullString    `json:"status" bigquery:"status"`
	Conclusion   bigquery.NullString    `json:"conclusion" bigquery:"conclusion"`
	StartedAt    bigquery.NullTimestamp `json:"started_at" bigquery:"started_at"`
	CompletedAt  bigquery.NullTimestamp `json:"completed_at" bigquery:"completed_at"`
	Name         bigquery.NullString    `json:"name" bigquery:"name"`
	CheckSuite   *CheckSuite            `json:"check_suite,omitempty" bigquery:"check_suite"`
	PullRequests []PullRequest          `json:"pull_requests,omitempty" bigquery:"pull_requests"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#CheckSuite
type CheckSuite struct {
	ID           bigquery.NullInt64     `json:"id" bigquery:"id"`
	HeadSHA      bigquery.NullString    `json:"head_sha" bigquery:"head_sha"`
	Status       bigquery.NullString    `json:"status" bigquery:"status"`
	Conclusion   bigquery.NullString    `json:"conclusion" bigquery:"conclusion"`
	CreatedAt    bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	UpdatedAt    bigquery.NullTimestamp `json:"updated_at" bigquery:"updated_at"`
	Repository   Repository             `json:"repository" bigquery:"repository"`
	PullRequests []PullRequest          `json:"pull_requests,omitempty" bigquery:"pull_requests"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#CheckSuiteEvent
type CheckSuiteEvent struct {
	Action       bigquery.NullString `json:"action" bigquery:"action"`
	CheckSuite   *CheckSuite         `json:"check_suite,omitempty" bigquery:"check_suite"`
	Repository   Repository          `json:"repository" bigquery:"repository"`
	Organization Organization        `json:"organization" bigquery:"organization"`
	Sender       User                `json:"sender" bigquery:"sender"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://github.com/google/go-github/blob/v60.0.0/github/event_types.go#L1085
type ProjectV2Item struct {
	ID            bigquery.NullInt64     `json:"id" bigquery:"id"`
	NodeID        bigquery.NullString    `json:"node_id" bigquery:"node_id"`
	ProjectNodeID bigquery.NullString    `json:"project_node_id" bigquery:"project_node_id"`
	ContentNodeID bigquery.NullString    `json:"content_node_id" bigquery:"content_node_id"`
	ContentType   bigquery.NullString    `json:"content_type" bigquery:"content_type"`
	Creator       *User                  `json:"creator,omitempty" bigquery:"creator"`
	CreatedAt     bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	UpdatedAt     bigquery.NullTimestamp `json:"updated_at" bigquery:"updated_at"`
	ArchivedAt    bigquery.NullTimestamp `json:"archived_at" bigquery:"archived_at"`
}

// https://github.com/google/go-github/blob/v60.0.0/github/event_types.go#L1062
type ProjectsV2ItemEvent struct {
	Action        bigquery.NullString `json:"action" bigquery:"action"`
	Changes       bigquery.NullJSON   `json:"changes" bigquery:"changes"`
	ProjectV2Item *ProjectV2Item      `json:"projects_v2_item,omitempty" bigquery:"projects_v2_item"`
	Organization  *Organization       `json:"organization,omitempty" bigquery:"organization"`
	Sender        *User               `json:"sender,omitempty" bigquery:"sender"`
	Installation  *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestReview
type PullRequestReview struct {
	ID                bigquery.NullInt64     `json:"id" bigquery:"id"`
	NodeID            bigquery.NullString    `json:"node_id" bigquery:"node_id"`
	User              User                   `json:"user" bigquery:"user"`
	Body              bigquery.NullString    `json:"body" bigquery:"body"`
	State             bigquery.NullString    `json:"state" bigquery:"state"`
	HTMLURL           bigquery.NullString    `json:"html_url" bigquery:"html_url"`
	PullRequestURL    bigquery.NullString    `json:"pull_request_url" bigquery:"pull_request_url"`
	SubmittedAt       bigquery.NullTimestamp `json:"submitted_at" bigquery:"submitted_at"`
	CommitID          bigquery.NullString    `json:"commit_id" bigquery:"commit_id"`
	AuthorAssociation bigquery.NullString    `json:"author_association" bigquery:"author_association"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestReviewEvent
type PullRequestReviewEvent struct {
	Action       bigquery.NullString `json:"action" bigquery:"action"`
	Review       PullRequestReview   `json:"review" bigquery:"review"`
	PullRequest  PullRequest         `json:"pull_request" bigquery:"pull_request"`
	Repository   Repository          `json:"repository" bigquery:"repository"`
	Organization Organization        `json:"organization" bigquery:"organization"`
	Sender       User                `json:"sender" bigquery:"sender"`
	Installation *Installation       `json:"installation,omitempty" bigquery:"installation"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestComment
type PullRequestReviewComment struct {
	ID                  bigquery.NullInt64     `json:"id" bigquery:"id"`
	NodeID              bigquery.NullString    `json:"node_id" bigquery:"node_id"`
	InReplyTo           bigquery.NullInt64     `json:"in_reply_to_id" bigquery:"in_reply_to_id"`
	Body                bigquery.NullString    `json:"body" bigquery:"body"`
	Path                bigquery.NullString    `json:"path" bigquery:"path"`
	DiffHunk            bigquery.NullString    `json:"diff_hunk" bigquery:"diff_hunk"`
	PullRequestReviewID bigquery.NullInt64     `json:"pull_request_review_id" bigquery:"pull_request_review_id"`
	Position            bigquery.NullInt64     `json:"position" bigquery:"position"`
	OriginalPosition    bigquery.NullInt64     `json:"original_position" bigquery:"original_position"`
	StartLine           bigquery.NullInt64     `json:"start_line" bigquery:"start_line"`
	Line                bigquery.NullInt64     `json:"line" bigquery:"line"`
	OriginalLine        bigquery.NullInt64     `json:"original_line" bigquery:"original_line"`
	OriginalStartLine   bigquery.NullInt64     `json:"original_start_line" bigquery:"original_start_line"`
	Side                bigquery.NullString    `json:"side" bigquery:"side"`
	StartSide           bigquery.NullString    `json:"start_side" bigquery:"start_side"`
	CommitID            bigquery.NullString    `json:"commit_id" bigquery:"commit_id"`
	OriginalCommitID    bigquery.NullString    `json:"original_commit_id" bigquery:"original_commit_id"`
	User                User                   `json:"user" bigquery:"user"`
	CreatedAt           bigquery.NullTimestamp `json:"created_at" bigquery:"created_at"`
	UpdatedAt           bigquery.NullTimestamp `json:"updated_at" bigquery:"updated_at"`
	AuthorAssociation   bigquery.NullString    `json:"author_association" bigquery:"author_association"`
	URL                 bigquery.NullString    `json:"url" bigquery:"url"`
	HTMLURL             bigquery.NullString    `json:"html_url" bigquery:"html_url"`
	PullRequestURL      bigquery.NullString    `json:"pull_request_url" bigquery:"pull_request_url"`
}

// https://pkg.go.dev/github.com/google/go-github/v60/github#PullRequestReviewCommentEvent
type PullRequestReviewCommentEvent struct {
	Action       bigquery.NullString      `json:"action" bigquery:"action"`
	Comment      PullRequestReviewComment `json:"comment" bigquery:"comment"`
	PullRequest  PullRequest              `json:"pull_request" bigquery:"pull_request"`
	Repository   Repository               `json:"repository" bigquery:"repository"`
	Organization Organization             `json:"organization" bigquery:"organization"`
	Sender       User                     `json:"sender" bigquery:"sender"`
	Installation *Installation            `json:"installation,omitempty" bigquery:"installation"`
}
