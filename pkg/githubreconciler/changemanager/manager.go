/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package changemanager

import (
	"context"
	"errors"
	"fmt"
	"text/template"

	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	internaltemplate "github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler/internal/template"
	"github.com/google/go-github/v75/github"
)

// Option configures a CM (ChangeManager).
type Option[T any] func(*CM[T])

// WithOwner overrides the GitHub owner (org or user) from the resource.
// When set, all PR operations will use this owner instead of the resource's owner.
func WithOwner[T any](owner string) Option[T] {
	return func(cm *CM[T]) {
		cm.owner = owner
	}
}

// WithRepo overrides the GitHub repository from the resource.
// When set, all PR operations will use this repo instead of the resource's repo.
func WithRepo[T any](repo string) Option[T] {
	return func(cm *CM[T]) {
		cm.repo = repo
	}
}

// CM manages the lifecycle of GitHub Pull Requests for a specific identity.
// It uses Go templates to generate PR titles and bodies from generic data of type T.
type CM[T any] struct {
	identity         string
	titleTemplate    *template.Template
	bodyTemplate     *template.Template
	templateExecutor *internaltemplate.Template[T]
	owner            string
	repo             string
}

// New creates a new CM with the given identity and templates.
// The templates are executed with data of type T when creating or updating PRs.
// Returns an error if titleTemplate or bodyTemplate is nil.
func New[T any](identity string, titleTemplate *template.Template, bodyTemplate *template.Template, opts ...Option[T]) (*CM[T], error) {
	if titleTemplate == nil {
		return nil, errors.New("titleTemplate cannot be nil")
	}
	if bodyTemplate == nil {
		return nil, errors.New("bodyTemplate cannot be nil")
	}

	templateExecutor, err := internaltemplate.New[T](identity, "-pr-data", "PR")
	if err != nil {
		return nil, fmt.Errorf("creating template executor: %w", err)
	}

	cm := &CM[T]{
		identity:         identity,
		titleTemplate:    titleTemplate,
		bodyTemplate:     bodyTemplate,
		templateExecutor: templateExecutor,
	}

	for _, opt := range opts {
		opt(cm)
	}

	return cm, nil
}

// NewSession creates a new Session for the given resource.
// It validates that the resource is a Path type and queries for any existing PRs
// with a head branch matching {identity}/{path}.
func (cm *CM[T]) NewSession(
	ctx context.Context,
	client *github.Client,
	res *githubreconciler.Resource,
) (*Session[T], error) {
	if res.Type != githubreconciler.ResourceTypePath {
		return nil, fmt.Errorf("change manager only supports Path resources, got: %v", res.Type)
	}

	// Determine which owner/repo to use
	owner := res.Owner
	repo := res.Repo
	if cm.owner != "" {
		owner = cm.owner
	}
	if cm.repo != "" {
		repo = cm.repo
	}

	branchName := cm.identity + "/" + res.Path
	headRef := owner + ":" + branchName

	// Query for existing PRs with this head branch
	prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		State: "open",
		Head:  headRef,
		Base:  res.Ref,
	})
	if err != nil {
		return nil, fmt.Errorf("listing pull requests: %w", err)
	}

	var existingPR *github.PullRequest
	if len(prs) > 0 {
		// Fetch the full PR details to populate fields like Mergeable
		// These fields are not populated by the List operation
		existingPR, _, err = client.PullRequests.Get(ctx, owner, repo, prs[0].GetNumber())
		if err != nil {
			return nil, fmt.Errorf("getting pull request #%d: %w", prs[0].GetNumber(), err)
		}
	}

	return &Session[T]{
		manager:    cm,
		client:     client,
		resource:   res,
		owner:      owner,
		repo:       repo,
		branchName: branchName,
		existingPR: existingPR,
	}, nil
}
