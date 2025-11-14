// Copyright 2025 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

package httpmetrics

import (
	"net/http"
	"regexp"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type pathPattern struct {
	pattern *regexp.Regexp
	bucket  string
}

// Default GitHub API endpoint patterns.
// Based on GitHub REST API documentation: https://docs.github.com/en/rest
var githubAPIPatterns = []pathPattern{{
	// https://docs.github.com/en/rest/repos/repos#get-a-repository
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+$`),
	bucket:  "/repos/{org}/{repo}",
}, {
	// https://docs.github.com/en/rest/issues/issues#list-repository-issues
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues$`),
	bucket:  "/repos/{org}/{repo}/issues",
}, {
	// https://docs.github.com/en/rest/issues/issues#get-an-issue
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues/\d+$`),
	bucket:  "/repos/{org}/{repo}/issues/{number}",
}, {
	// https://docs.github.com/en/rest/issues/comments#list-issue-comments
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues/\d+/comments$`),
	bucket:  "/repos/{org}/{repo}/issues/{number}/comments",
}, {
	// https://docs.github.com/en/rest/issues/labels#add-labels-to-an-issue
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/issues/\d+/labels$`),
	bucket:  "/repos/{org}/{repo}/issues/{number}/labels",
}, {
	// https://docs.github.com/en/rest/pulls/pulls#list-pull-requests
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/pulls$`),
	bucket:  "/repos/{org}/{repo}/pulls",
}, {
	// https://docs.github.com/en/rest/pulls/pulls#get-a-pull-request
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/pulls/\d+$`),
	bucket:  "/repos/{org}/{repo}/pulls/{number}",
}, {
	// https://docs.github.com/en/rest/pulls/pulls#list-pull-requests-files
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/pulls/\d+/files$`),
	bucket:  "/repos/{org}/{repo}/pulls/{number}/files",
}, {
	// https://docs.github.com/en/rest/pulls/comments#list-review-comments-on-a-pull-request
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/pulls/\d+/comments$`),
	bucket:  "/repos/{org}/{repo}/pulls/{number}/comments",
}, {
	// https://docs.github.com/en/rest/pulls/reviews#list-reviews-for-a-pull-request
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/pulls/\d+/reviews$`),
	bucket:  "/repos/{org}/{repo}/pulls/{number}/reviews",
}, {
	// https://docs.github.com/en/rest/commits/commits#list-commits
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/commits$`),
	bucket:  "/repos/{org}/{repo}/commits",
}, {
	// https://docs.github.com/en/rest/commits/commits#get-a-commit
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/commits/[^/]+$`),
	bucket:  "/repos/{org}/{repo}/commits/{sha}",
}, {
	// https://docs.github.com/en/rest/commits/statuses#get-the-combined-status-for-a-specific-reference
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/commits/[^/]+/status$`),
	bucket:  "/repos/{org}/{repo}/commits/{sha}/status",
}, {
	// https://docs.github.com/en/rest/commits/statuses#create-a-commit-status
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/statuses/[^/]+$`),
	bucket:  "/repos/{org}/{repo}/statuses/{sha}",
}, {
	// https://docs.github.com/en/rest/repos/contents#get-repository-content
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/contents/.*$`),
	bucket:  "/repos/{org}/{repo}/contents/{path}",
}, {
	// https://docs.github.com/en/rest/branches/branches#list-branches
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/branches$`),
	bucket:  "/repos/{org}/{repo}/branches",
}, {
	// https://docs.github.com/en/rest/branches/branches#get-a-branch
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/branches/[^/]+$`),
	bucket:  "/repos/{org}/{repo}/branches/{branch}",
}, {
	// https://docs.github.com/en/rest/repos/repos#list-repository-tags
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/tags$`),
	bucket:  "/repos/{org}/{repo}/tags",
}, {
	// https://docs.github.com/en/rest/releases/releases#list-releases
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/releases$`),
	bucket:  "/repos/{org}/{repo}/releases",
}, {
	// https://docs.github.com/en/rest/releases/releases#get-a-release
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/releases/\d+$`),
	bucket:  "/repos/{org}/{repo}/releases/{id}",
}, {
	// https://docs.github.com/en/rest/actions/workflow-runs#list-workflow-runs-for-a-repository
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/actions/runs$`),
	bucket:  "/repos/{org}/{repo}/actions/runs",
}, {
	// https://docs.github.com/en/rest/actions/workflow-runs#get-a-workflow-run
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/actions/runs/\d+$`),
	bucket:  "/repos/{org}/{repo}/actions/runs/{id}",
}, {
	// https://docs.github.com/en/rest/actions/workflows#list-repository-workflows
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/actions/workflows$`),
	bucket:  "/repos/{org}/{repo}/actions/workflows",
}, {
	// https://docs.github.com/en/rest/actions/workflows#get-a-workflow
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/actions/workflows/[^/]+$`),
	bucket:  "/repos/{org}/{repo}/actions/workflows/{id}",
}, {
	// https://docs.github.com/en/rest/checks/runs#create-a-check-run
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/check-runs$`),
	bucket:  "/repos/{org}/{repo}/check-runs",
}, {
	// https://docs.github.com/en/rest/checks/runs#get-a-check-run
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/check-runs/\d+$`),
	bucket:  "/repos/{org}/{repo}/check-runs/{id}",
}, {
	// https://docs.github.com/en/rest/checks/runs#list-check-runs-for-a-git-reference
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/commits/[^/]+/check-runs$`),
	bucket:  "/repos/{org}/{repo}/commits/{ref}/check-runs",
}, {
	// https://docs.github.com/en/rest/commits/commits#list-branches-for-head-commit
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/commits/[^/]+/branches-where-head$`),
	bucket:  "/repos/{org}/{repo}/commits/{sha}/branches-where-head",
}, {
	// https://docs.github.com/en/rest/commits/commits#list-pull-requests-associated-with-a-commit
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/commits/[^/]+/pulls$`),
	bucket:  "/repos/{org}/{repo}/commits/{sha}/pulls",
}, {
	// https://docs.github.com/en/rest/branches/branches#rename-a-branch
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/branches/[^/]+/rename$`),
	bucket:  "/repos/{org}/{repo}/branches/{branch}/rename",
}, {
	// https://docs.github.com/en/rest/branches/branches#merge-a-branch
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/merges$`),
	bucket:  "/repos/{org}/{repo}/merges",
}, {
	// https://docs.github.com/en/rest/pulls/pulls#check-if-a-pull-request-has-been-merged
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/pulls/\d+/merge$`),
	bucket:  "/repos/{org}/{repo}/pulls/{number}/merge",
}, {
	// https://docs.github.com/en/rest/pulls/pulls#update-a-pull-request-branch
	pattern: regexp.MustCompile(`^/repos/[^/]+/[^/]+/pulls/\d+/update-branch$`),
	bucket:  "/repos/{org}/{repo}/pulls/{number}/update-branch",
}, {
	// https://docs.github.com/en/rest/orgs/orgs#get-an-organization
	pattern: regexp.MustCompile(`^/orgs/[^/]+$`),
	bucket:  "/orgs/{org}",
}, {
	// https://docs.github.com/en/rest/repos/repos#list-organization-repositories
	pattern: regexp.MustCompile(`^/orgs/[^/]+/repos$`),
	bucket:  "/orgs/{org}/repos",
}, {
	// https://docs.github.com/en/rest/orgs/members#list-organization-members
	pattern: regexp.MustCompile(`^/orgs/[^/]+/members$`),
	bucket:  "/orgs/{org}/members",
}, {
	// https://docs.github.com/en/rest/teams/teams#list-teams
	pattern: regexp.MustCompile(`^/orgs/[^/]+/teams$`),
	bucket:  "/orgs/{org}/teams",
}, {
	// https://docs.github.com/en/rest/users/users#get-the-authenticated-user
	pattern: regexp.MustCompile(`^/user$`),
	bucket:  "/user",
}, {
	// https://docs.github.com/en/rest/repos/repos#list-repositories-for-the-authenticated-user
	pattern: regexp.MustCompile(`^/user/repos$`),
	bucket:  "/user/repos",
}, {
	// https://docs.github.com/en/rest/users/users#get-a-user
	pattern: regexp.MustCompile(`^/users/[^/]+$`),
	bucket:  "/users/{user}",
}, {
	// https://docs.github.com/en/rest/repos/repos#list-repositories-for-a-user
	pattern: regexp.MustCompile(`^/users/[^/]+/repos$`),
	bucket:  "/users/{user}/repos",
}}

func bucketizePath(path string) string {
	for _, p := range githubAPIPatterns {
		if p.pattern.MatchString(path) {
			return p.bucket
		}
	}
	return ""
}

func instrumentGitHubAPI(next http.RoundTripper) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		return next.RoundTrip(r)
	}
}
