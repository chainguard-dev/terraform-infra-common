package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
	"github.com/google/go-github/v61/github"
)

// This bot is responsible for passing/failing a check run based on the presence of any "blocking/*" label.
//
// This check can be made required so that blockers block merging of PRs.

func New() sdk.Bot { return bot{} }

type bot struct{}

func (b bot) Name() string { return "blocker" }

func (b bot) OnPullRequest(ctx context.Context, pr *github.PullRequest) error {
	log := clog.FromContext(ctx)
	owner, repo := *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name

	hasBlockingLabel := slices.ContainsFunc(pr.Labels, func(l *github.Label) bool {
		return strings.HasPrefix(*l.Name, "blocking/")
	})

	client := sdk.NewGitHubClient(ctx, owner, repo, b.Name())
	defer client.Close(ctx)

	conclusion := "success"
	if hasBlockingLabel {
		conclusion = "failure"
		log.Debug("PR %d has blocking label")
	}
	sha := *pr.Head.SHA

	// If there are no check runs for the current head SHA, create one with the conclusion.
	crs, _, err := client.Client().Checks.ListCheckRunsForRef(ctx, owner, repo, sha, &github.ListCheckRunsOptions{CheckName: github.String(b.Name())})
	if err != nil {
		return fmt.Errorf("listing check runs: %w", err)
	}
	if len(crs.CheckRuns) == 0 {
		log.Infof("Creating CheckRun for PR %d sha %s with conclusion %s", *pr.Number, sha, conclusion)
		if _, _, err := client.Client().Checks.CreateCheckRun(ctx, owner, repo, github.CreateCheckRunOptions{
			Name:       b.Name(),
			HeadSHA:    sha,
			Status:     github.String("completed"),
			Conclusion: &conclusion,
		}); err != nil {
			return fmt.Errorf("creating check run: %w", err)
		}
		return nil
	}

	// No change, nothing else to do.
	if *crs.CheckRuns[0].Conclusion == conclusion {
		log.Debugf("CheckRun for PR %d sha %s already has conclusion %s", *pr.Number, sha, conclusion)
		return nil
	}

	// If there's already a check run, update its conclusion.
	log.Infof("Updating CheckRun for PR %d sha %s with conclusion %s", *pr.Number, sha, conclusion)
	if _, _, err = client.Client().Checks.UpdateCheckRun(ctx, owner, repo, *crs.CheckRuns[0].ID, github.UpdateCheckRunOptions{
		Name:       b.Name(),
		Status:     github.String("completed"),
		Conclusion: &conclusion,
	}); err != nil {
		return fmt.Errorf("updating check run %s: %w", *crs.CheckRuns[0].ID, err)
	}
	return nil
}
