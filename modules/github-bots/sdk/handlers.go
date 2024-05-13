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

type IssueCommentHandler func(ctx context.Context, ice github.IssueCommentEvent) error

func (r IssueCommentHandler) EventType() EventType {
	return IssueCommentEvent
}

type WorkflowRunArtifactHandler func(ctx context.Context, wre github.WorkflowRunEvent) error

func (r WorkflowRunArtifactHandler) EventType() EventType {
	return WorkflowRunArtifactEvent
}

type EventType string

const (
	// Github events (https://github.com/chainguard-dev/terraform-infra-common/tree/main/modules/github-events)
	PullRequestEvent  EventType = "dev.chainguard.github.pull_request"
	WorkflowRunEvent  EventType = "dev.chainguard.github.workflow_run"
	IssueCommentEvent EventType = "dev.chainguard.github.issue_comment"

	// LoFo events
	WorkflowRunArtifactEvent EventType = "dev.chainguard.lofo.workflow_run_artifacts"
)
