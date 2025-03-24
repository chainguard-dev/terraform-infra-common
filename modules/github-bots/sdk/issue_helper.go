package sdk

import (
	"fmt"
	"strings"

	"github.com/google/go-github/v68/github"
)

// getIssueRepoInfo extracts owner and repository name from an issue
// the repository object is not available on the issue, so we need to extract it from the URL
func getIssueRepoInfo(issue *github.Issue) (owner, repoName string, err error) {
	// If the issue has a repository object, we can get the owner and repo name from there
	if issue.Repository != nil {
		owner = issue.Repository.GetOwner().GetLogin()
		repoName = issue.Repository.GetName()
		return
	}

	// If the repository object is not available, we need to extract it from the URL
	// Split the URL by "/"
	parts := strings.Split(issue.GetRepositoryURL(), "/")

	// URLs should be in format https://api.github.com/repos/owner/repo
	// We need at least 2 parts after "repos" to get owner and repo
	if len(parts) >= 2 {
		// Get the last two parts
		repoName = parts[len(parts)-1]
		owner = parts[len(parts)-2]
		return
	}

	// If we don't have at least 2 parts, return an error
	return "", "", fmt.Errorf("found %d parts in URL %s, expected at least 2", len(parts), issue.GetRepositoryURL())
}
