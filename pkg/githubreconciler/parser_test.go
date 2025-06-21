/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"reflect"
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *Resource
		wantErr bool
	}{
		{
			name: "valid issue URL",
			url:  "https://github.com/owner/repo/issues/123",
			want: &Resource{
				Owner:  "owner",
				Repo:   "repo",
				Type:   ResourceTypeIssue,
				Number: 123,
				URL:    "https://github.com/owner/repo/issues/123",
			},
		},
		{
			name: "valid pull request URL",
			url:  "https://github.com/owner/repo/pull/456",
			want: &Resource{
				Owner:  "owner",
				Repo:   "repo",
				Type:   ResourceTypePullRequest,
				Number: 456,
				URL:    "https://github.com/owner/repo/pull/456",
			},
		},
		{
			name:    "invalid URL - www.github.com",
			url:     "https://www.github.com/owner/repo/issues/789",
			wantErr: true,
		},
		{
			name: "owner with hyphen",
			url:  "https://github.com/my-owner/repo/issues/1",
			want: &Resource{
				Owner:  "my-owner",
				Repo:   "repo",
				Type:   ResourceTypeIssue,
				Number: 1,
				URL:    "https://github.com/my-owner/repo/issues/1",
			},
		},
		{
			name: "repo with dots and hyphens",
			url:  "https://github.com/owner/my.complex-repo.name/pull/42",
			want: &Resource{
				Owner:  "owner",
				Repo:   "my.complex-repo.name",
				Type:   ResourceTypePullRequest,
				Number: 42,
				URL:    "https://github.com/owner/my.complex-repo.name/pull/42",
			},
		},
		{
			name:    "invalid URL - wrong host",
			url:     "https://gitlab.com/owner/repo/issues/123",
			wantErr: true,
		},
		{
			name:    "invalid URL - no issue/PR type",
			url:     "https://github.com/owner/repo/123",
			wantErr: true,
		},
		{
			name:    "invalid URL - no number",
			url:     "https://github.com/owner/repo/issues",
			wantErr: true,
		},
		{
			name:    "invalid URL - non-numeric number",
			url:     "https://github.com/owner/repo/issues/abc",
			wantErr: true,
		},
		{
			name:    "invalid URL - missing owner",
			url:     "https://github.com/repo/issues/123",
			wantErr: true,
		},
		{
			name:    "invalid URL - empty string",
			url:     "",
			wantErr: true,
		},
		{
			name:    "invalid URL - not a URL",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "invalid URL - wrong resource type",
			url:     "https://github.com/owner/repo/commits/123",
			wantErr: true,
		},
		{
			name:    "invalid URL - too many path segments",
			url:     "https://github.com/owner/repo/issues/123/comments",
			wantErr: true,
		},
		{
			name:    "invalid URL - too few path segments",
			url:     "https://github.com/owner",
			wantErr: true,
		},
		{
			name: "http URL still works",
			url:  "http://github.com/owner/repo/issues/123",
			want: &Resource{
				Owner:  "owner",
				Repo:   "repo",
				Type:   ResourceTypeIssue,
				Number: 123,
				URL:    "http://github.com/owner/repo/issues/123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResource_String(t *testing.T) {
	tests := []struct {
		name     string
		resource *Resource
		want     string
	}{
		{
			name: "issue",
			resource: &Resource{
				Owner:  "owner",
				Repo:   "repo",
				Type:   ResourceTypeIssue,
				Number: 123,
			},
			want: "owner/repo#123",
		},
		{
			name: "pull request",
			resource: &Resource{
				Owner:  "owner",
				Repo:   "repo",
				Type:   ResourceTypePullRequest,
				Number: 456,
			},
			want: "owner/repo#456",
		},
		{
			name: "complex names",
			resource: &Resource{
				Owner:  "my-org",
				Repo:   "my.complex-repo",
				Type:   ResourceTypeIssue,
				Number: 789,
			},
			want: "my-org/my.complex-repo#789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resource.String(); got != tt.want {
				t.Errorf("Resource.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseURL_ValidatesURL(t *testing.T) {
	// Test that URL field is preserved
	urls := []string{
		"https://github.com/owner/repo/issues/123",
		"https://github.com/owner/repo/pull/456",
		"https://github.com/my-org/my.repo/issues/789",
	}

	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			resource, err := ParseURL(url)
			if err != nil {
				t.Fatalf("ParseURL() error = %v", err)
			}
			if resource.URL != url {
				t.Errorf("Resource.URL = %v, want %v", resource.URL, url)
			}
		})
	}
}
