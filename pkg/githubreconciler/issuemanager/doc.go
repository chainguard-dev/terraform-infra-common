/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package issuemanager provides an abstraction for managing GitHub Issue
// lifecycle operations, similar to how the changemanager handles GitHub Pull Requests.
//
// The IssueManager encapsulates common issue patterns including:
//   - Finding existing issues by labels
//   - Embedding/extracting structured data in issue bodies
//   - Creating or updating multiple issues (upsert pattern)
//   - Matching existing issues to desired issues via custom matcher function
//   - Checking for skip labels
//   - Closing any outstanding issues not in the desired set
//
// # Key Differences from ChangeManager
//
// Unlike ChangeManager which assumes one PR per resource path, IssueManager
// handles multiple issues per resource path:
//   - Session tracks multiple existing issues instead of a single PR
//   - UpsertMany handles a slice of desired issue states
//   - Custom matcher function determines which existing issues match desired ones
//   - Queries use labels (e.g., "identity:path") to find relevant issues
//   - No branch or code change concepts since issues don't require branches
//
// # Generic Type Parameter
//
// IssueManager is generic over a type T that represents structured data to be
// embedded in issue bodies. This data is used to:
//   - Execute Go templates for issue titles and bodies
//   - Determine if an issue needs to be refreshed
//   - Match existing issues to desired issues
//   - Store metadata about the issue's purpose
//
// # Usage
//
// Create an IssueManager once per identity with parsed templates:
//
//	titleTmpl, _ := template.New("title").Parse(`CVE-{{.CVE_ID}} in {{.PackageName}}`)
//	bodyTmpl, _ := template.New("body").Parse(`Vulnerability {{.CVE_ID}} found in {{.PackageName}} {{.Version}}`)
//
//	im := issuemanager.New[MyData]("security-bot", titleTmpl, bodyTmpl)
//
// Optionally, include label templates to generate dynamic labels from issue data:
//
//	labelTmpl1, _ := template.New("severity").Parse(`severity:{{.Severity}}`)
//	labelTmpl2, _ := template.New("package").Parse(`package:{{.PackageName}}`)
//
//	im := issuemanager.New[MyData]("security-bot", titleTmpl, bodyTmpl, labelTmpl1, labelTmpl2)
//
// Create a session per reconciliation:
//
//	session, err := im.NewSession(ctx, ghClient, resource)
//	if err != nil {
//	    return err
//	}
//
// Check for skip labels:
//
//	if session.HasSkipLabel() {
//	    return nil
//	}
//
// Define a matcher function to identify which issues correspond to which data:
//
//	matcher := func(a, b MyData) bool {
//	    return a.CVE_ID == b.CVE_ID && a.PackageName == b.PackageName
//	}
//
// Upsert multiple issues with data:
//
//	issueURLs, err := session.UpsertMany(ctx, []*MyData{data1, data2, data3}, matcher, labels)
//
// Close any outstanding issues not in the desired set:
//
//	if err := session.CloseAnyOutstanding(ctx, []*MyData{data1, data2, data3}, matcher, "Closing resolved issue"); err != nil {
//	    return err
//	}
package issuemanager
