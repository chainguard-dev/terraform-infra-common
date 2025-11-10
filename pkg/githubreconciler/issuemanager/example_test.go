/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package issuemanager_test

import (
	"context"
	"text/template"

	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler"
	"github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler/issuemanager"
	"github.com/google/go-github/v75/github"
)

type VulnerabilityData struct {
	CVEID       string
	PackageName string
	Version     string
	Severity    string
}

func Example() {
	// Parse templates once at initialization
	titleTmpl := template.Must(template.New("title").Parse(`CVE-{{.CVEID}}: {{.Severity}} vulnerability in {{.PackageName}}`))
	bodyTmpl := template.Must(template.New("body").Parse(`Vulnerability **{{.CVEID}}** detected in {{.PackageName}} {{.Version}}

**Severity**: {{.Severity}}

Please update to a patched version.`))

	// Optional: Define label templates to generate dynamic labels from issue data
	labelTmpl1 := template.Must(template.New("severity").Parse(`severity:{{.Severity}}`))
	labelTmpl2 := template.Must(template.New("package").Parse(`package:{{.PackageName}}`))

	// Create manager once per identity with label templates
	im, err := issuemanager.New[VulnerabilityData]("security-bot", titleTmpl, bodyTmpl, labelTmpl1, labelTmpl2)
	if err != nil {
		// handle error
		return
	}

	// In your reconciler, create a session per resource
	ctx := context.Background()
	var ghClient *github.Client // your GitHub client
	var res *githubreconciler.Resource

	session, err := im.NewSession(ctx, ghClient, res)
	if err != nil {
		// handle error
		return
	}

	// Check for skip label
	if session.HasSkipLabel() {
		// skip this resource
		return
	}

	// Define desired issues with data
	desired := []*VulnerabilityData{
		{
			CVEID:       "2024-1234",
			PackageName: "openssl",
			Version:     "3.0.0",
			Severity:    "HIGH",
		},
		{
			CVEID:       "2024-5678",
			PackageName: "curl",
			Version:     "8.0.0",
			Severity:    "MEDIUM",
		},
	}

	// Define a matcher function to identify issues
	matcher := func(a, b VulnerabilityData) bool {
		return a.CVEID == b.CVEID && a.PackageName == b.PackageName
	}

	// Upsert multiple issues
	_, err = session.UpsertMany(ctx, desired, matcher, []string{"security", "automated"})
	if err != nil {
		// handle error
		return
	}

	// Close any issues that are no longer in the desired set
	err = session.CloseAnyOutstanding(ctx, desired, matcher, "Vulnerability has been resolved")
	if err != nil {
		// handle error
		return
	}
}
