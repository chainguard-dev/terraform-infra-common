package main

import (
	"context"
	"fmt"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
	"github.com/google/go-github/v61/github"
)

type bot struct{}

func New() sdk.Bot { return bot{} }

func (b bot) Name() string { return "time" }

func (b bot) OnPullRequest(ctx context.Context, pr *github.PullRequest) error {
	owner, repo := *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name

	client := sdk.NewGitHubClient(ctx, owner, repo, b.Name())
	defer client.Close(ctx)

	return client.SetComment(ctx, pr, b.Name(), fmt.Sprintf("The time is now %s", time.Now().Format(time.RFC3339)))
}
