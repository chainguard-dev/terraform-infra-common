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

type ExampleData struct {
	Foo string
	Bar string
	Baz string
}

// Equal implements the Comparable interface for ExampleData.
// It compares Foo and Bar to determine if two instances represent the same issue.
func (e ExampleData) Equal(other ExampleData) bool {
	return e.Foo == other.Foo && e.Bar == other.Bar
}

func Example() {
	// Parse templates once at initialization
	titleTmpl := template.Must(template.New("title").Parse(`Issue {{.Foo}}: {{.Baz}}`))
	bodyTmpl := template.Must(template.New("body").Parse(`This is issue **{{.Foo}}** for {{.Bar}}

**Status**: {{.Baz}}

Additional details here.`))

	// Optional: Define label templates to generate dynamic labels from issue data
	labelTmpl1 := template.Must(template.New("label1").Parse(`status:{{.Baz}}`))
	labelTmpl2 := template.Must(template.New("label2").Parse(`category:{{.Bar}}`))

	// Create manager once per identity with label templates
	im, err := issuemanager.New[ExampleData]("example-manager", titleTmpl, bodyTmpl,
		issuemanager.WithLabelTemplates[ExampleData](labelTmpl1, labelTmpl2),
	)
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

	// Define desired issues with data
	// Note: Issues with skip labels (skip:example-manager) will be automatically preserved
	desired := []*ExampleData{{
		Foo: "foo",
		Bar: "bar",
		Baz: "baz",
	}, {
		Foo: "bar",
		Bar: "baz",
		Baz: "foo",
	}}

	// Reconcile performs a complete reconciliation: create, update, and close operations
	// Matching is done using the Equal method
	_, err = session.Reconcile(ctx, desired, []string{"example", "automated"}, "Issue no longer relevant")
	if err != nil {
		// handle error
		return
	}
}
