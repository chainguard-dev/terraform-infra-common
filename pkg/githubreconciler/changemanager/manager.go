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
	"github.com/google/go-github/v75/github"
)

// CM manages the lifecycle of GitHub Pull Requests for a specific identity.
// It uses Go templates to generate PR titles and bodies from generic data of type T.
type CM[T any] struct {
	identity      string
	titleTemplate *template.Template
	bodyTemplate  *template.Template
}

// New creates a new CM with the given identity and templates.
// The templates are executed with data of type T when creating or updating PRs.
// Returns an error if titleTemplate or bodyTemplate is nil.
func New[T any](identity string, titleTemplate *template.Template, bodyTemplate *template.Template) (*CM[T], error) {
	if titleTemplate == nil {
		return nil, errors.New("titleTemplate cannot be nil")
	}
	if bodyTemplate == nil {
		return nil, errors.New("bodyTemplate cannot be nil")
	}

	return &CM[T]{
		identity:      identity,
		titleTemplate: titleTemplate,
		bodyTemplate:  bodyTemplate,
	}, nil
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

	branchName := cm.identity + "/" + res.Path
	headRef := res.Owner + ":" + branchName

	// Query for existing PRs with this head branch
	prs, _, err := client.PullRequests.List(ctx, res.Owner, res.Repo, &github.PullRequestListOptions{
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
		existingPR, _, err = client.PullRequests.Get(ctx, res.Owner, res.Repo, prs[0].GetNumber())
		if err != nil {
			return nil, fmt.Errorf("getting pull request #%d: %w", prs[0].GetNumber(), err)
		}
	}

	return &Session[T]{
		manager:    cm,
		client:     client,
		resource:   res,
		branchName: branchName,
		existingPR: existingPR,
	}, nil
}
