//go:build integration
// +build integration

package sdk

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-github/v61/github"
)

// NOTE: This is an integration test that requires 'GITHUB_TOKEN' env variable to be set!
// It is recommended to run this test in a local environment.
func Test_SearchFilenameInRepository(t *testing.T) {
	ctx := context.Background()

	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Fatalf("GITHUB_TOKEN env var not set\n")
	}

	// create a GitHub client
	repoOrg := "kserve"
	repoName := "kserve"
	// sdk allows an override of GIT_TOKEN env var so we can test from a local environment
	cli := NewGitHubClient(ctx, repoOrg, repoName, "test")
	result, err := cli.SearchFilenameInRepository(ctx, repoOrg, repoName, "pyproject.toml", &github.ListOptions{})
	if err != nil {
		t.Fatalf("SearchFilenameInRepository err: %v\n", err)
	}

	if *result.Total == 0 {
		t.Fatalf("SearchFilenameInRepository result is zero\n")
	}
}
