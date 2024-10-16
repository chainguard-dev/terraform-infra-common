package main

import (
	"context"
	"strings"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
	"github.com/google/go-github/v61/github"
)

const label = "blocking/dnm"

func New() sdk.Bot {
	name := "dnm"

	handler := sdk.PullRequestHandler(func(ctx context.Context, pre github.PullRequestEvent) error {
		cli := sdk.NewGitHubClient(ctx, *pre.Repo.Owner.Login, *pre.Repo.Name, name)
		defer cli.Close(ctx)

		// If the title contains some variant of "dnm" and the PR doesn't have the label, add it -- this will no-op if it already has it.
		for _, dnm := range []string{name, "do not merge", "donotmerge", "do-not-merge"} {
			if strings.Contains(strings.ToLower(*pre.PullRequest.Title), dnm) {
				return cli.AddLabel(ctx, pre.PullRequest, label)
			}
		}

		// If it has the label and the title doesn't match, remove the label -- this will no-op if it already doesn't have it.
		return cli.RemoveLabel(ctx, pre.PullRequest, label)
	})

	return sdk.NewBot(name,
		sdk.BotWithHandler(handler),
	)
}
