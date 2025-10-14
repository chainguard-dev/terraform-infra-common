/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package patterns

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
		// For success cases, test that specific paths match/don't match
		matchPaths   []string
		noMatchPaths []string
	}{{
		name:  "single pattern matches all files",
		input: `["(.+)"]`,
		matchPaths: []string{
			"file.txt",
			"dir/file.yaml",
			"deep/nested/path/file.go",
		},
		noMatchPaths: []string{
			"",
		},
	}, {
		name:  "pattern matches yaml files only",
		input: `["(.+\\.yaml)"]`,
		matchPaths: []string{
			"config.yaml",
			"dir/config.yaml",
			"deep/nested/file.yaml",
		},
		noMatchPaths: []string{
			"file.txt",
			"file.yml",
			"yaml",
		},
	}, {
		name:  "pattern matches specific directory",
		input: `["(infrastructure/.+)"]`,
		matchPaths: []string{
			"infrastructure/main.tf",
			"infrastructure/nested/file.go",
		},
		noMatchPaths: []string{
			"main.tf",
			"src/infrastructure/main.tf",
			"infrastructure",
		},
	}, {
		name:  "multiple patterns",
		input: `["(.+\\.yaml)", "(configs/.+)"]`,
		matchPaths: []string{
			"file.yaml",
			"dir/file.yaml",
			"configs/app.json",
			"configs/nested/config.toml",
		},
		noMatchPaths: []string{
			"file.txt",
			"other/file.txt",
		},
	}, {
		name:        "invalid JSON",
		input:       `not json`,
		wantErr:     true,
		errContains: "failed to parse patterns JSON",
	}, {
		name:        "empty array",
		input:       `[]`,
		wantErr:     true,
		errContains: "no valid patterns found",
	}, {
		name:        "pattern with no capture group",
		input:       `[".+\\.yaml"]`,
		wantErr:     true,
		errContains: "must have exactly one capture group, got 0",
	}, {
		name:        "pattern with multiple capture groups",
		input:       `["(.+)/(.+)"]`,
		wantErr:     true,
		errContains: "must have exactly one capture group, got 2",
	}, {
		name:        "invalid regex",
		input:       `["(invalid["]`,
		wantErr:     true,
		errContains: "invalid regex",
	}, {
		name:  "pattern without explicit anchors gets them added",
		input: `["(test.*)"]`,
		matchPaths: []string{
			"test",
			"test123",
			"testing",
			"test-suffix-extra",
		},
		noMatchPaths: []string{
			"prefix-test",
		},
	}, {
		name:  "pattern extracts correct capture group",
		input: `["dir/(.+\\.go)"]`,
		matchPaths: []string{
			"dir/main.go",
			"dir/test.go",
			"dir/nested/main.go",
		},
		noMatchPaths: []string{
			"main.go",
			"other/main.go",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns, err := Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Parse() expected error, got nil")
				}
				if tt.errContains != "" {
					if got := err.Error(); !strings.Contains(got, tt.errContains) {
						t.Errorf("Parse() error: got = %q, wanted to contain = %q", got, tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}

			if len(patterns) == 0 {
				t.Fatal("Parse() returned empty patterns slice")
			}

			// Test that expected paths match
			for _, path := range tt.matchPaths {
				matched := false
				for _, pattern := range patterns {
					if pattern.MatchString(path) {
						matched = true
						break
					}
				}
				if !matched {
					t.Errorf("expected path %q to match one of the patterns, but it didn't", path)
				}
			}

			// Test that unexpected paths don't match
			for _, path := range tt.noMatchPaths {
				for i, pattern := range patterns {
					if pattern.MatchString(path) {
						t.Errorf("expected path %q not to match pattern[%d] %q, but it did", path, i, pattern.String())
					}
				}
			}
		})
	}
}

func TestParseCaptureGroup(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		testPath     string
		wantCaptured string
	}{{
		name:         "captures entire path",
		input:        `["(.+)"]`,
		testPath:     "dir/file.txt",
		wantCaptured: "dir/file.txt",
	}, {
		name:         "captures filename only",
		input:        `[".+/([^/]+)"]`,
		testPath:     "dir/file.txt",
		wantCaptured: "file.txt",
	}, {
		name:         "captures yaml files",
		input:        `["(.+\\.yaml)"]`,
		testPath:     "config/app.yaml",
		wantCaptured: "config/app.yaml",
	}, {
		name:         "captures path within directory",
		input:        `["infrastructure/(.+)"]`,
		testPath:     "infrastructure/modules/vpc.tf",
		wantCaptured: "modules/vpc.tf",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}

			if len(patterns) == 0 {
				t.Fatal("Parse() returned empty patterns slice")
			}

			matches := patterns[0].FindStringSubmatch(tt.testPath)
			if len(matches) < 2 {
				t.Fatalf("pattern did not match path %q", tt.testPath)
			}

			captured := matches[1]
			if captured != tt.wantCaptured {
				t.Errorf("captured group: got = %q, wanted = %q", captured, tt.wantCaptured)
			}
		})
	}
}

func TestParseAnchoring(t *testing.T) {
	// Test that anchors are always added unconditionally
	input := `["(.+\\.yaml)"]`
	patterns, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}

	pattern := patterns[0].String()

	// Check that the compiled pattern has anchors
	if pattern[0] != '^' {
		t.Errorf("pattern should start with ^, got: %s", pattern)
	}
	if pattern[len(pattern)-1] != '$' {
		t.Errorf("pattern should end with $, got: %s", pattern)
	}

	// Verify it matches complete paths with .yaml extension
	testCases := []struct {
		path        string
		shouldMatch bool
	}{
		{"file.yaml", true},
		{"dir/file.yaml", true},
		{"prefix-file.yaml", true},
		{"file.yaml-extra", false},
		{"file.txt", false},
	}

	for _, tc := range testCases {
		matched := patterns[0].MatchString(tc.path)
		if matched != tc.shouldMatch {
			t.Errorf("path %q: got match = %v, wanted = %v", tc.path, matched, tc.shouldMatch)
		}
	}
}
