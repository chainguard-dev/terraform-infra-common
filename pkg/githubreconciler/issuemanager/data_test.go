/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package issuemanager

import (
	"strings"
	"testing"
	"text/template"
)

type testData struct {
	CVEID       string
	PackageName string
	Version     string
	Severity    string
}

func Test_embedData(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("CVE-{{.CVEID}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.PackageName}} {{.Version}}"))
	im, err := New[testData]("security-bot", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := &testData{
		CVEID:       "2024-1234",
		PackageName: "openssl",
		Version:     "3.0.0",
		Severity:    "HIGH",
	}

	body := "This is the issue body"
	embedded, err := im.embedData(body, data)
	if err != nil {
		t.Fatalf("embedData failed: %v", err)
	}

	// Verify the original body is present
	if !strings.Contains(embedded, body) {
		t.Errorf("embedded body missing original content")
	}

	// Verify the markers are present
	if !strings.Contains(embedded, "<!--security-bot-issue-data-->") {
		t.Errorf("embedded body missing start marker")
	}
	if !strings.Contains(embedded, "<!--/security-bot-issue-data-->") {
		t.Errorf("embedded body missing end marker")
	}

	// Verify we can extract the data back
	extracted, err := im.extractData(embedded)
	if err != nil {
		t.Fatalf("extractData failed: %v", err)
	}

	if extracted.CVEID != data.CVEID {
		t.Errorf("CVEID: got = %q, wanted = %q", extracted.CVEID, data.CVEID)
	}
	if extracted.PackageName != data.PackageName {
		t.Errorf("PackageName: got = %q, wanted = %q", extracted.PackageName, data.PackageName)
	}
	if extracted.Version != data.Version {
		t.Errorf("Version: got = %q, wanted = %q", extracted.Version, data.Version)
	}
	if extracted.Severity != data.Severity {
		t.Errorf("Severity: got = %q, wanted = %q", extracted.Severity, data.Severity)
	}
}

func Test_extractData_notFound(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("CVE-{{.CVEID}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.PackageName}}"))
	im, err := New[testData]("security-bot", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	body := "This is an issue body without embedded data"
	_, err = im.extractData(body)
	if err == nil {
		t.Error("extractData should have failed for body without embedded data")
	}
}

func Test_executeTemplate(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("CVE-{{.CVEID}} in {{.PackageName}}"))
	bodyTmpl := template.Must(template.New("body").Parse("Update"))
	im, err := New[testData]("security-bot", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := &testData{
		CVEID:       "2024-1234",
		PackageName: "openssl",
	}

	result, err := im.executeTemplate(titleTmpl, data)
	if err != nil {
		t.Fatalf("executeTemplate failed: %v", err)
	}

	expected := "CVE-2024-1234 in openssl"
	if result != expected {
		t.Errorf("template result: got = %q, wanted = %q", result, expected)
	}
}

func TestNew(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("CVE-{{.CVEID}}"))
	bodyTmpl := template.Must(template.New("body").Parse("Vulnerability in {{.PackageName}}"))

	tests := []struct {
		name          string
		identity      string
		titleTemplate *template.Template
		bodyTemplate  *template.Template
		wantErr       bool
		errContains   string
	}{{
		name:          "valid templates",
		identity:      "security-bot",
		titleTemplate: titleTmpl,
		bodyTemplate:  bodyTmpl,
		wantErr:       false,
	}, {
		name:          "nil title template",
		identity:      "security-bot",
		titleTemplate: nil,
		bodyTemplate:  bodyTmpl,
		wantErr:       true,
		errContains:   "titleTemplate cannot be nil",
	}, {
		name:          "nil body template",
		identity:      "security-bot",
		titleTemplate: titleTmpl,
		bodyTemplate:  nil,
		wantErr:       true,
		errContains:   "bodyTemplate cannot be nil",
	}, {
		name:          "both templates nil",
		identity:      "security-bot",
		titleTemplate: nil,
		bodyTemplate:  nil,
		wantErr:       true,
		errContains:   "titleTemplate cannot be nil",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im, err := New[testData](tt.identity, tt.titleTemplate, tt.bodyTemplate)
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
				if im == nil {
					t.Fatal("New() returned nil IM when error is nil")
				}
				if im.identity != tt.identity {
					t.Errorf("New() identity: got = %q, wanted = %q", im.identity, tt.identity)
				}
			}
		})
	}
}

func TestNew_withLabelTemplates(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("CVE-{{.CVEID}}"))
	bodyTmpl := template.Must(template.New("body").Parse("Vulnerability in {{.PackageName}}"))
	labelTmpl1 := template.Must(template.New("severity").Parse("severity:{{.Severity}}"))
	labelTmpl2 := template.Must(template.New("package").Parse("package:{{.PackageName}}"))

	im, err := New[testData]("security-bot", titleTmpl, bodyTmpl, labelTmpl1, labelTmpl2)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if im == nil {
		t.Fatal("New() returned nil IM")
	}

	if len(im.labelTemplates) != 2 {
		t.Errorf("labelTemplates length: got = %d, wanted = 2", len(im.labelTemplates))
	}
}

func Test_executeLabelTemplates(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("CVE-{{.CVEID}}"))
	bodyTmpl := template.Must(template.New("body").Parse("Update"))
	labelTmpl1 := template.Must(template.New("severity").Parse("severity:{{.Severity}}"))
	labelTmpl2 := template.Must(template.New("package").Parse("package:{{.PackageName}}"))

	im, err := New[testData]("security-bot", titleTmpl, bodyTmpl, labelTmpl1, labelTmpl2)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := &testData{
		CVEID:       "2024-1234",
		PackageName: "openssl",
		Severity:    "HIGH",
	}

	// Test executing label templates
	label1, err := im.executeTemplate(labelTmpl1, data)
	if err != nil {
		t.Fatalf("executeTemplate for severity failed: %v", err)
	}
	if label1 != "severity:HIGH" {
		t.Errorf("severity label: got = %q, wanted = %q", label1, "severity:HIGH")
	}

	label2, err := im.executeTemplate(labelTmpl2, data)
	if err != nil {
		t.Fatalf("executeTemplate for package failed: %v", err)
	}
	if label2 != "package:openssl" {
		t.Errorf("package label: got = %q, wanted = %q", label2, "package:openssl")
	}
}
