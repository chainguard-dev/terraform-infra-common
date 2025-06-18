package githubbot

import (
	"context"
	"encoding/json"

	"github.com/google/go-github/v72/github"
)

type EventHandlerFunc interface {
	EventType() EventType
}

type PullRequestHandler func(ctx context.Context, pre github.PullRequestEvent) error

func (r PullRequestHandler) EventType() EventType {
	return PullRequestEvent
}

type WorkflowRunHandler func(ctx context.Context, wre github.WorkflowRunEvent) error

func (r WorkflowRunHandler) EventType() EventType {
	return WorkflowRunEvent
}

type WorkflowRunLogsHandler func(ctx context.Context, wre github.WorkflowRunEvent) error

func (r WorkflowRunLogsHandler) EventType() EventType {
	return WorkflowRunLogsEvent
}

type IssuesHandler func(ctx context.Context, ice github.IssueEvent) error

func (r IssuesHandler) EventType() EventType {
	return IssuesEvent
}

type IssueCommentHandler func(ctx context.Context, ice github.IssueCommentEvent) error

func (r IssueCommentHandler) EventType() EventType {
	return IssueCommentEvent
}

type PushHandler func(ctx context.Context, pre github.PushEvent) error

func (r PushHandler) EventType() EventType {
	return PushEvent
}

type WorkflowRunArtifactHandler func(ctx context.Context, wre github.WorkflowRunEvent) error

func (r WorkflowRunArtifactHandler) EventType() EventType {
	return WorkflowRunArtifactEvent
}

type CheckRunHandler func(ctx context.Context, pre github.CheckRunEvent) error

func (r CheckRunHandler) EventType() EventType {
	return CheckRunEvent
}

type CheckSuiteHandler func(ctx context.Context, pre github.CheckSuiteEvent) error

func (r CheckSuiteHandler) EventType() EventType {
	return CheckSuiteEvent
}

type ProjectsV2ItemHandler func(ctx context.Context, pie ProjectsV2ItemEvent) error

func (r ProjectsV2ItemHandler) EventType() EventType {
	return ProjectsV2ItemEventType
}

// https://github.com/google/go-github/blob/v60.0.0/github/event_types.go#L1062
//
// ProjectsV2ItemEvent represents a project_v2_item event. It's copied from go-github since
// their version only supports the `archived` action.
type ProjectsV2ItemEvent struct {
	Action        string               `json:"action,omitempty"`
	Changes       json.RawMessage      `json:"changes,omitempty"`
	ProjectV2Item *ProjectV2Item       `json:"projects_v2_item,omitempty"`
	Organization  *github.Organization `json:"organization,omitempty"`
	Sender        *github.User         `json:"sender,omitempty"`
}

// https://github.com/google/go-github/blob/v60.0.0/github/event_types.go#L1085
type ProjectV2Item struct {
	ID            int64             `json:"id,omitempty"`
	NodeID        string            `json:"node_id,omitempty"`
	ProjectNodeID string            `json:"project_node_id,omitempty"`
	ContentNodeID string            `json:"content_node_id,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	CreatedAt     *github.Timestamp `json:"created_at,omitempty"`
	UpdatedAt     *github.Timestamp `json:"updated_at,omitempty"`
	ArchivedAt    *github.Timestamp `json:"archived_at,omitempty"`
}

type EventType string

const (
	// GitHub events (https://github.com/chainguard-dev/terraform-infra-common/tree/main/modules/github-events)
	PullRequestEvent        EventType = "dev.chainguard.github.pull_request"
	WorkflowRunEvent        EventType = "dev.chainguard.github.workflow_run"
	IssuesEvent             EventType = "dev.chainguard.github.issues"
	IssueCommentEvent       EventType = "dev.chainguard.github.issue_comment"
	PushEvent               EventType = "dev.chainguard.github.push"
	CheckRunEvent           EventType = "dev.chainguard.github.check_run"
	CheckSuiteEvent         EventType = "dev.chainguard.github.check_suite"
	ProjectsV2ItemEventType EventType = "dev.chainguard.github.projects_v2_item"

	// LoFo events
	WorkflowRunArtifactEvent EventType = "dev.chainguard.lofo.workflow_run_artifacts"
	WorkflowRunLogsEvent     EventType = "dev.chainguard.lofo.workflow_run_logs"
)
