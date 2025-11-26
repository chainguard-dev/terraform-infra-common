/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package changemanager

import (
	"strings"
	"testing"
	"text/template"
)

type testData struct {
	PackageName string
	Version     string
	Commit      string
}

func TestNew(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.PackageName}}/{{.Version}}"))
	bodyTmpl := template.Must(template.New("body").Parse("Update {{.PackageName}} to {{.Version}}"))

	tests := []struct {
		name          string
		identity      string
		titleTemplate *template.Template
		bodyTemplate  *template.Template
		wantErr       bool
		errContains   string
	}{{
		name:          "valid templates",
		identity:      "test-bot",
		titleTemplate: titleTmpl,
		bodyTemplate:  bodyTmpl,
		wantErr:       false,
	}, {
		name:          "nil title template",
		identity:      "test-bot",
		titleTemplate: nil,
		bodyTemplate:  bodyTmpl,
		wantErr:       true,
		errContains:   "titleTemplate cannot be nil",
	}, {
		name:          "nil body template",
		identity:      "test-bot",
		titleTemplate: titleTmpl,
		bodyTemplate:  nil,
		wantErr:       true,
		errContains:   "bodyTemplate cannot be nil",
	}, {
		name:          "both templates nil",
		identity:      "test-bot",
		titleTemplate: nil,
		bodyTemplate:  nil,
		wantErr:       true,
		errContains:   "titleTemplate cannot be nil",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := New[testData](tt.identity, tt.titleTemplate, tt.bodyTemplate)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error: got = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Error("New() should have returned an error")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("New() error message: got = %q, wanted to contain %q", err.Error(), tt.errContains)
				}
			} else {
				if cm == nil {
					t.Fatal("New() returned nil CM when error is nil")
					return
				}
				if cm.identity != tt.identity {
					t.Errorf("New() identity: got = %q, wanted = %q", cm.identity, tt.identity)
				}
			}
		})
	}
}
