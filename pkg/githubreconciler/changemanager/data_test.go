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
	cm := New[testData]("test-bot", nil, nil)

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
	cm := New[testData]("test-bot", nil, nil)

	body := "This is a PR body without embedded data"
	_, err := cm.extractData(body)
	if err == nil {
		t.Error("extractData should have failed for body without embedded data")
	}
}

func Test_executeTemplate(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.PackageName}}/{{.Version}}"))
	cm := New[testData]("test-bot", titleTmpl, nil)

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
