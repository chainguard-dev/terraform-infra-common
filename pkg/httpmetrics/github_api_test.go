// Copyright 2025 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

package httpmetrics

import (
	"testing"
)

func Test_bucketizePath(t *testing.T) {
	tests := []struct {
		path   string
		bucket string
	}{{
		path:   "/repos/chainguard-dev/terraform-infra-common",
		bucket: "/repos/{org}/{repo}",
	}, {
		path:   "/repos/octocat/hello-world/issues",
		bucket: "/repos/{org}/{repo}/issues",
	}, {
		path:   "/repos/octocat/hello-world/issues/42",
		bucket: "/repos/{org}/{repo}/issues/{number}",
	}, {
		path:   "/repos/octocat/hello-world/issues/123/comments",
		bucket: "/repos/{org}/{repo}/issues/{number}/comments",
	}, {
		path:   "/repos/octocat/hello-world/pulls",
		bucket: "/repos/{org}/{repo}/pulls",
	}, {
		path:   "/repos/octocat/hello-world/pulls/1",
		bucket: "/repos/{org}/{repo}/pulls/{number}",
	}, {
		path:   "/repos/octocat/hello-world/pulls/1/files",
		bucket: "/repos/{org}/{repo}/pulls/{number}/files",
	}, {
		path:   "/repos/octocat/hello-world/pulls/1/comments",
		bucket: "/repos/{org}/{repo}/pulls/{number}/comments",
	}, {
		path:   "/repos/octocat/hello-world/pulls/1/reviews",
		bucket: "/repos/{org}/{repo}/pulls/{number}/reviews",
	}, {
		path:   "/repos/octocat/hello-world/commits",
		bucket: "/repos/{org}/{repo}/commits",
	}, {
		path:   "/repos/octocat/hello-world/commits/abc123",
		bucket: "/repos/{org}/{repo}/commits/{sha}",
	}, {
		path:   "/repos/octocat/hello-world/commits/abc123/status",
		bucket: "/repos/{org}/{repo}/commits/{sha}/status",
	}, {
		path:   "/repos/octocat/hello-world/statuses/abc123",
		bucket: "/repos/{org}/{repo}/statuses/{sha}",
	}, {
		path:   "/repos/octocat/hello-world/contents/README.md",
		bucket: "/repos/{org}/{repo}/contents/{path}",
	}, {
		path:   "/repos/octocat/hello-world/contents/path/to/file.go",
		bucket: "/repos/{org}/{repo}/contents/{path}",
	}, {
		path:   "/repos/octocat/hello-world/branches",
		bucket: "/repos/{org}/{repo}/branches",
	}, {
		path:   "/repos/octocat/hello-world/branches/main",
		bucket: "/repos/{org}/{repo}/branches/{branch}",
	}, {
		path:   "/repos/octocat/hello-world/tags",
		bucket: "/repos/{org}/{repo}/tags",
	}, {
		path:   "/repos/octocat/hello-world/releases",
		bucket: "/repos/{org}/{repo}/releases",
	}, {
		path:   "/repos/octocat/hello-world/releases/42",
		bucket: "/repos/{org}/{repo}/releases/{id}",
	}, {
		path:   "/repos/octocat/hello-world/actions/runs",
		bucket: "/repos/{org}/{repo}/actions/runs",
	}, {
		path:   "/repos/octocat/hello-world/actions/runs/123",
		bucket: "/repos/{org}/{repo}/actions/runs/{id}",
	}, {
		path:   "/repos/octocat/hello-world/actions/workflows",
		bucket: "/repos/{org}/{repo}/actions/workflows",
	}, {
		path:   "/repos/octocat/hello-world/actions/workflows/build.yml",
		bucket: "/repos/{org}/{repo}/actions/workflows/{id}",
	}, {
		path:   "/orgs/chainguard-dev",
		bucket: "/orgs/{org}",
	}, {
		path:   "/orgs/chainguard-dev/repos",
		bucket: "/orgs/{org}/repos",
	}, {
		path:   "/orgs/chainguard-dev/members",
		bucket: "/orgs/{org}/members",
	}, {
		path:   "/orgs/chainguard-dev/teams",
		bucket: "/orgs/{org}/teams",
	}, {
		path:   "/user",
		bucket: "/user",
	}, {
		path:   "/user/repos",
		bucket: "/user/repos",
	}, {
		path:   "/users/octocat",
		bucket: "/users/{user}",
	}, {
		path:   "/users/octocat/repos",
		bucket: "/users/{user}/repos",
	}, {
		path:   "/some/unknown/path",
		bucket: "",
	}}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := bucketizePath(tt.path)
			if got != tt.bucket {
				t.Errorf("bucketizePath() = %v, want = %v", got, tt.bucket)
			}
		})
	}
}


