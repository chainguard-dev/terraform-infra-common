/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package changemanager provides an abstraction for managing GitHub Pull Request
// lifecycle operations, similar to how the statusmanager handles GitHub Check Runs.
//
// The ChangeManager encapsulates common PR patterns including:
//   - Finding existing PRs by head branch
//   - Embedding/extracting structured data in PR bodies
//   - Creating or updating PRs (upsert pattern)
//   - Checking for skip labels
//   - Closing any outstanding PRs
//
// # Generic Type Parameter
//
// Like the Status Manager, ChangeManager is generic over a type T that represents
// structured data to be embedded in PR bodies. This data is used to:
//   - Execute Go templates for PR titles and bodies
//   - Determine if a PR needs to be refreshed
//   - Store metadata about the PR's purpose
//
// # Usage
//
// Create a ChangeManager once per identity with parsed templates:
//
//	titleTmpl, _ := template.New("title").Parse(`{{.PackageName}}/{{.Version}} update`)
//	bodyTmpl, _ := template.New("body").Parse(`Update {{.PackageName}} to {{.Version}}`)
//
//	cm := changemanager.New[MyData]("update-bot", titleTmpl, bodyTmpl)
//
// Create a session per reconciliation:
//
//	session, err := cm.NewSession(ctx, ghClient, resource)
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
// Upsert a PR with data:
//
//	prURL, err := session.Upsert(ctx, data, draft, labels, func(ctx context.Context, branchName string) error {
//	    // Make code changes on the branch
//	    return makeChanges(ctx, branchName)
//	})
//
// Close any outstanding PRs with a message:
//
//	if err := session.CloseAnyOutstanding(ctx, "Closing due to version downgrade"); err != nil {
//	    return err
//	}
package changemanager
