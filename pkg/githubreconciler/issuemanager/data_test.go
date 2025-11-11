/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package issuemanager

import (
	"testing"
	"text/template"
)

type testData struct {
	VulnID      string
	PackageName string
	Version     string
	Severity    string
}

func Test_executeLabelTemplates(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.VulnID}}"))
	bodyTmpl := template.Must(template.New("body").Parse("Update"))
	labelTmpl1 := template.Must(template.New("severity").Parse("severity:{{.Severity}}"))
	labelTmpl2 := template.Must(template.New("package").Parse("package:{{.PackageName}}"))

	im, err := New[testData]("security-bot", titleTmpl, bodyTmpl, labelTmpl1, labelTmpl2)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := &testData{
		VulnID:      "2024-1234",
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
