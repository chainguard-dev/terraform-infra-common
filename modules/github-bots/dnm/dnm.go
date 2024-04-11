package main

import (
	"context"
	"strings"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
	"github.com/google/go-github/v61/github"
)

type bot struct{}

func New() sdk.Bot { return bot{} }

const label = "blocking/dnm"

func (b bot) Name() string { return "dnm" }

func (b bot) OnPullRequest(ctx context.Context, pr *github.PullRequest) error {
	owner, repo := *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name

	client := sdk.NewGitHubClient(ctx, owner, repo, b.Name())
	defer client.Close(ctx)

	// If the title contains some variant of "dnm" and the PR doesn't have the label, add it -- this will no-op if it already has it.
	for _, dnm := range []string{"dnm", "do not merge", "donotmerge", "do-not-merge"} {
		if strings.Contains(strings.ToLower(*pr.Title), dnm) {
			return client.AddLabel(ctx, pr, label)
		}
	}

	// If it has the label and the title doesn't match, remove the label -- this will no-op if it already doesn't have it.
	return client.RemoveLabel(ctx, pr, label)
}
