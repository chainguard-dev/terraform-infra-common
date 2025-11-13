// Copyright 2025 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

package httpmetrics

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/chainguard-dev/clog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type pathPattern struct {
	pattern *regexp.Regexp
	bucket  string
}

var githubAPIPatterns = []pathPattern{}

func init() {
	// Default GitHub API endpoint patterns.
	// Based on GitHub REST API documentation: https://docs.github.com/en/rest
	patterns := []struct {
		regex  string
		bucket string
	}{{
		// https://docs.github.com/en/rest/repos/repos#get-a-repository
		regex:  `^/repos/[^/]+/[^/]+$`,
		bucket: "/repos/{org}/{repo}",
	}, {
		// https://docs.github.com/en/rest/issues/issues#list-repository-issues
		regex:  `^/repos/[^/]+/[^/]+/issues$`,
		bucket: "/repos/{org}/{repo}/issues",
	}, {
		// https://docs.github.com/en/rest/issues/issues#get-an-issue
		regex:  `^/repos/[^/]+/[^/]+/issues/\d+$`,
		bucket: "/repos/{org}/{repo}/issues/{number}",
	}, {
		// https://docs.github.com/en/rest/issues/comments#list-issue-comments
		regex:  `^/repos/[^/]+/[^/]+/issues/\d+/comments$`,
		bucket: "/repos/{org}/{repo}/issues/{number}/comments",
	}, {
		// https://docs.github.com/en/rest/pulls/pulls#list-pull-requests
		regex:  `^/repos/[^/]+/[^/]+/pulls$`,
		bucket: "/repos/{org}/{repo}/pulls",
	}, {
		// https://docs.github.com/en/rest/pulls/pulls#get-a-pull-request
		regex:  `^/repos/[^/]+/[^/]+/pulls/\d+$`,
		bucket: "/repos/{org}/{repo}/pulls/{number}",
	}, {
		// https://docs.github.com/en/rest/pulls/pulls#list-pull-requests-files
		regex:  `^/repos/[^/]+/[^/]+/pulls/\d+/files$`,
		bucket: "/repos/{org}/{repo}/pulls/{number}/files",
	}, {
		// https://docs.github.com/en/rest/pulls/comments#list-review-comments-on-a-pull-request
		regex:  `^/repos/[^/]+/[^/]+/pulls/\d+/comments$`,
		bucket: "/repos/{org}/{repo}/pulls/{number}/comments",
	}, {
		// https://docs.github.com/en/rest/pulls/reviews#list-reviews-for-a-pull-request
		regex:  `^/repos/[^/]+/[^/]+/pulls/\d+/reviews$`,
		bucket: "/repos/{org}/{repo}/pulls/{number}/reviews",
	}, {
		// https://docs.github.com/en/rest/commits/commits#list-commits
		regex:  `^/repos/[^/]+/[^/]+/commits$`,
		bucket: "/repos/{org}/{repo}/commits",
	}, {
		// https://docs.github.com/en/rest/commits/commits#get-a-commit
		regex:  `^/repos/[^/]+/[^/]+/commits/[^/]+$`,
		bucket: "/repos/{org}/{repo}/commits/{sha}",
	}, {
		// https://docs.github.com/en/rest/commits/statuses#get-the-combined-status-for-a-specific-reference
		regex:  `^/repos/[^/]+/[^/]+/commits/[^/]+/status$`,
		bucket: "/repos/{org}/{repo}/commits/{sha}/status",
	}, {
		// https://docs.github.com/en/rest/commits/statuses#create-a-commit-status
		regex:  `^/repos/[^/]+/[^/]+/statuses/[^/]+$`,
		bucket: "/repos/{org}/{repo}/statuses/{sha}",
	}, {
		// https://docs.github.com/en/rest/repos/contents#get-repository-content
		regex:  `^/repos/[^/]+/[^/]+/contents/.*$`,
		bucket: "/repos/{org}/{repo}/contents/{path}",
	}, {
		// https://docs.github.com/en/rest/branches/branches#list-branches
		regex:  `^/repos/[^/]+/[^/]+/branches$`,
		bucket: "/repos/{org}/{repo}/branches",
	}, {
		// https://docs.github.com/en/rest/branches/branches#get-a-branch
		regex:  `^/repos/[^/]+/[^/]+/branches/[^/]+$`,
		bucket: "/repos/{org}/{repo}/branches/{branch}",
	}, {
		// https://docs.github.com/en/rest/repos/repos#list-repository-tags
		regex:  `^/repos/[^/]+/[^/]+/tags$`,
		bucket: "/repos/{org}/{repo}/tags",
	}, {
		// https://docs.github.com/en/rest/releases/releases#list-releases
		regex:  `^/repos/[^/]+/[^/]+/releases$`,
		bucket: "/repos/{org}/{repo}/releases",
	}, {
		// https://docs.github.com/en/rest/releases/releases#get-a-release
		regex:  `^/repos/[^/]+/[^/]+/releases/\d+$`,
		bucket: "/repos/{org}/{repo}/releases/{id}",
	}, {
		// https://docs.github.com/en/rest/actions/workflow-runs#list-workflow-runs-for-a-repository
		regex:  `^/repos/[^/]+/[^/]+/actions/runs$`,
		bucket: "/repos/{org}/{repo}/actions/runs",
	}, {
		// https://docs.github.com/en/rest/actions/workflow-runs#get-a-workflow-run
		regex:  `^/repos/[^/]+/[^/]+/actions/runs/\d+$`,
		bucket: "/repos/{org}/{repo}/actions/runs/{id}",
	}, {
		// https://docs.github.com/en/rest/actions/workflows#list-repository-workflows
		regex:  `^/repos/[^/]+/[^/]+/actions/workflows$`,
		bucket: "/repos/{org}/{repo}/actions/workflows",
	}, {
		// https://docs.github.com/en/rest/actions/workflows#get-a-workflow
		regex:  `^/repos/[^/]+/[^/]+/actions/workflows/[^/]+$`,
		bucket: "/repos/{org}/{repo}/actions/workflows/{id}",
	}, {
		// https://docs.github.com/en/rest/orgs/orgs#get-an-organization
		regex:  `^/orgs/[^/]+$`,
		bucket: "/orgs/{org}",
	}, {
		// https://docs.github.com/en/rest/repos/repos#list-organization-repositories
		regex:  `^/orgs/[^/]+/repos$`,
		bucket: "/orgs/{org}/repos",
	}, {
		// https://docs.github.com/en/rest/orgs/members#list-organization-members
		regex:  `^/orgs/[^/]+/members$`,
		bucket: "/orgs/{org}/members",
	}, {
		// https://docs.github.com/en/rest/teams/teams#list-teams
		regex:  `^/orgs/[^/]+/teams$`,
		bucket: "/orgs/{org}/teams",
	}, {
		// https://docs.github.com/en/rest/users/users#get-the-authenticated-user
		regex:  `^/user$`,
		bucket: "/user",
	}, {
		// https://docs.github.com/en/rest/repos/repos#list-repositories-for-the-authenticated-user
		regex:  `^/user/repos$`,
		bucket: "/user/repos",
	}, {
		// https://docs.github.com/en/rest/users/users#get-a-user
		regex:  `^/users/[^/]+$`,
		bucket: "/users/{user}",
	}, {
		// https://docs.github.com/en/rest/repos/repos#list-repositories-for-a-user
		regex:  `^/users/[^/]+/repos$`,
		bucket: "/users/{user}/repos",
	}}

	for _, p := range patterns {
		re, err := regexp.Compile(p.regex)
		if err != nil {
			panic(fmt.Sprintf("failed to compile pattern %q: %v", p.regex, err))
		}
		githubAPIPatterns = append(githubAPIPatterns, pathPattern{
			pattern: re,
			bucket:  p.bucket,
		})
	}
}

var seenPathMap = sync.Map{}

func bucketizePath(ctx context.Context, path string) string {
	for _, p := range githubAPIPatterns {
		if p.pattern.MatchString(path) {
			return p.bucket
		}
	}

	// Only log every 10th occurrence of an unknown path.
	v, _ := seenPathMap.LoadOrStore(path, &atomic.Int64{})
	vInt := v.(*atomic.Int64)
	if seen := vInt.Add(1); (seen-1)%10 == 0 {
		clog.WarnContext(ctx, `bucketing GitHub API path as "other"`, "path", path, "seen", seen)
	}

	return "other"
}

func instrumentGitHubAPI(next http.RoundTripper) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.github.com" {
			return next.RoundTrip(r)
		}

		endpoint := bucketizePath(r.Context(), r.URL.Path)
		ctx := withEndpoint(r.Context(), endpoint)
		r = r.WithContext(ctx)

		return next.RoundTrip(r)
	}
}
