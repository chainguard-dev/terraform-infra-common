package sdk

import (
	"context"

	"github.com/google/go-github/v61/github"
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

type EventType string

const (
	// GitHub events (https://github.com/chainguard-dev/terraform-infra-common/tree/main/modules/github-events)
	PullRequestEvent  EventType = "dev.chainguard.github.pull_request"
	WorkflowRunEvent  EventType = "dev.chainguard.github.workflow_run"
	IssueCommentEvent EventType = "dev.chainguard.github.issue_comment"
	PushEvent         EventType = "dev.chainguard.github.push"
	CheckRunEvent     EventType = "dev.chainguard.github.check_run"
	CheckSuiteEvent   EventType = "dev.chainguard.github.check_suite"

	// LoFo events
	WorkflowRunArtifactEvent EventType = "dev.chainguard.lofo.workflow_run_artifacts"
	WorkflowRunLogsEvent     EventType = "dev.chainguard.lofo.workflow_run_logs"
)
