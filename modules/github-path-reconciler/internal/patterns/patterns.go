/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package patterns

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

// Parse parses a JSON array of regex patterns and validates that each has exactly one capture group.
func Parse(patternsStr string) ([]*regexp.Regexp, error) {
	var patternStrings []string
	if err := json.Unmarshal([]byte(patternsStr), &patternStrings); err != nil {
		return nil, fmt.Errorf("failed to parse patterns JSON: %w", err)
	}

	patterns := make([]*regexp.Regexp, 0, len(patternStrings))

	for _, patternStr := range patternStrings {
		// Add implicit ^ and $ anchors unconditionally
		patternStr = "^" + patternStr + "$"

		regex, err := regexp.Compile(patternStr)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", patternStr, err)
		}

		// Ensure it has exactly one capture group
		numCaptures := regex.NumSubexp()
		if numCaptures != 1 {
			return nil, fmt.Errorf("regex %q must have exactly one capture group, got %d", patternStr, numCaptures)
		}

		patterns = append(patterns, regex)
	}

	if len(patterns) == 0 {
		return nil, errors.New("no valid patterns found")
	}

	return patterns, nil
}
