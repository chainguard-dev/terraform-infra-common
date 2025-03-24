package sdk

import (
	"testing"

	"github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/assert"
)

func Test_RepoInfo(t *testing.T) {
	t.Run("extract from issue with URL", func(t *testing.T) {
		// Create an issue with a repository URL directly
		issue := &github.Issue{
			RepositoryURL: github.Ptr("https://api.github.com/repos/foo/bar"),
		}

		org, repo, err := getIssueRepoInfo(issue)
		assert.Nil(t, err)
		assert.Equal(t, "foo", org)
		assert.Equal(t, "bar", repo)
	})

	t.Run("extract from issue with Repository object", func(t *testing.T) {
		// Create an issue with Repository object
		owner := &github.User{Login: github.Ptr("test-owner")}
		repo := &github.Repository{
			Name:  github.Ptr("test-repo"),
			Owner: owner,
		}
		issue := &github.Issue{
			Repository: repo,
		}

		org, repoName, err := getIssueRepoInfo(issue)
		assert.Nil(t, err)
		assert.Equal(t, "test-owner", org)
		assert.Equal(t, "test-repo", repoName)
	})

	t.Run("error case with malformed URL", func(t *testing.T) {
		// URL with fewer than 2 parts when split by "/"
		issue := &github.Issue{
			RepositoryURL: github.Ptr("no-slashes"),
		}

		org, repo, err := getIssueRepoInfo(issue)
		assert.NotNil(t, err, "Expected an error but got nil")
		assert.Empty(t, org, "Expected empty owner string")
		assert.Empty(t, repo, "Expected empty repo string")
	})
}
