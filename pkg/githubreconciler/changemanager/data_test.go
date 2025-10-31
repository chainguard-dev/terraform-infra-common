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

func Test_embedData(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.PackageName}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.Version}}"))
	cm, err := New[testData]("test-bot", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := &testData{
		PackageName: "foo",
		Version:     "1.2.3",
		Commit:      "abc123",
	}

	body := "This is the PR body"
	embedded, err := cm.embedData(body, data)
	if err != nil {
		t.Fatalf("embedData failed: %v", err)
	}

	// Verify the original body is present
	if !strings.Contains(embedded, body) {
		t.Errorf("embedded body missing original content")
	}

	// Verify the markers are present
	if !strings.Contains(embedded, "<!--test-bot-pr-data-->") {
		t.Errorf("embedded body missing start marker")
	}
	if !strings.Contains(embedded, "<!--/test-bot-pr-data-->") {
		t.Errorf("embedded body missing end marker")
	}

	// Verify we can extract the data back
	extracted, err := cm.extractData(embedded)
	if err != nil {
		t.Fatalf("extractData failed: %v", err)
	}

	if extracted.PackageName != data.PackageName {
		t.Errorf("PackageName: got = %q, wanted = %q", extracted.PackageName, data.PackageName)
	}
	if extracted.Version != data.Version {
		t.Errorf("Version: got = %q, wanted = %q", extracted.Version, data.Version)
	}
	if extracted.Commit != data.Commit {
		t.Errorf("Commit: got = %q, wanted = %q", extracted.Commit, data.Commit)
	}
}

func Test_extractData_notFound(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.PackageName}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.Version}}"))
	cm, err := New[testData]("test-bot", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	body := "This is a PR body without embedded data"
	_, err = cm.extractData(body)
	if err == nil {
		t.Error("extractData should have failed for body without embedded data")
	}
}

func Test_executeTemplate(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.PackageName}}/{{.Version}}"))
	bodyTmpl := template.Must(template.New("body").Parse("Update"))
	cm, err := New[testData]("test-bot", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := &testData{
		PackageName: "foo",
		Version:     "1.2.3",
	}

	result, err := cm.executeTemplate(titleTmpl, data)
	if err != nil {
		t.Fatalf("executeTemplate failed: %v", err)
	}

	expected := "foo/1.2.3"
	if result != expected {
		t.Errorf("template result: got = %q, wanted = %q", result, expected)
	}
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
				}
				if cm.identity != tt.identity {
					t.Errorf("New() identity: got = %q, wanted = %q", cm.identity, tt.identity)
				}
			}
		})
	}
}
