/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseURL parses a GitHub issue or pull request URL and extracts the components.
//
// Expected formats:
//   - https://github.com/org/repo/issues/123
//   - https://github.com/org/repo/pull/123
func ParseURL(uri string) (*Resource, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Validate host
	if parsed.Host != "github.com" {
		return nil, fmt.Errorf("invalid host: %s (expected github.com)", parsed.Host)
	}

	// Split path into components
	// Expected: /org/repo/issues/123 or /org/repo/pull/123
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid path format: %s", parsed.Path)
	}

	owner := parts[0]
	repo := parts[1]
	resourceType := parts[2]
	numberStr := parts[3]

	// Parse number
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return nil, fmt.Errorf("invalid resource number: %s", numberStr)
	}

	// Determine resource type
	var resType ResourceType
	switch resourceType {
	case "issues":
		resType = ResourceTypeIssue
	case "pull":
		resType = ResourceTypePullRequest
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	return &Resource{
		Owner:  owner,
		Repo:   repo,
		Number: number,
		Type:   resType,
		URL:    uri,
	}, nil
}
