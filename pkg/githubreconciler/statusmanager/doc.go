/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package statusmanager provides Kubernetes-style status management for GitHub reconcilers
// using GitHub Check Runs API.
//
// This package implements a pattern similar to Kubernetes reconciliation where:
// - Each commit SHA is treated like a Kubernetes resource generation
// - The reconciler tracks which SHA it has observed/processed
// - Status is persisted in GitHub Check Runs instead of PR comments
// - Check Runs provide per-commit status tracking with structured data
//
// # Architecture
//
// The package is built around three main types:
//
// StatusManager: The top-level manager that holds configuration like identity,
// project ID, and service name. It creates sessions for individual PR reconciliations.
//
// Session: Represents a reconciliation session for a specific PR. It encapsulates
// all the state needed to read and write status for a particular commit SHA.
//
// Status: The structured data that gets persisted in Check Runs. It includes
// the observed generation (SHA), status, conclusion, and reconciler-specific details.
//
// # Usage
//
// First, create a StatusManager with your reconciler's identity:
//
//	sm, err := statusmanager.NewStatusManager[MyDetails](ctx, "my-reconciler")
//	if err != nil {
//	    return err
//	}
//
// For each PR reconciliation, create a session:
//
//	session := sm.NewSession(githubClient, pullRequest)
//
// Check the current state (optional):
//
//	status, err := session.ObservedState(ctx)
//	if err != nil {
//	    return err
//	}
//	// Use status to determine if reconciliation is needed
//
// Update status during reconciliation:
//
//	// Set initial in-progress status
//	err = session.SetActualState(ctx, "Processing", &statusmanager.Status[MyDetails]{
//	    Status: "in_progress",
//	    Details: MyDetails{},
//	})
//
//	// ... perform reconciliation ...
//
//	// Set final status
//	err = session.SetActualState(ctx, "Complete", &statusmanager.Status[MyDetails]{
//	    Status: "completed",
//	    Conclusion: "success",
//	    Details: MyDetails{
//	        // ... reconciliation results ...
//	    },
//	})
//
// # Custom Markdown Rendering
//
// If your details type implements the Markdown() method, it will be used
// to generate visible output in the Check Run:
//
//	type MyDetails struct {
//	    FilesProcessed int
//	    Errors []string
//	}
//
//	func (d MyDetails) Markdown() string {
//	    return fmt.Sprintf("Processed %d files", d.FilesProcessed)
//	}
//
// # Status Embedding
//
// The structured status data is embedded in HTML comments within the Check Run
// output, making it both human-readable (via Markdown) and machine-parseable
// (via embedded JSON). This allows the reconciler to read back its previous
// state on subsequent runs.
//
// # Cloud Logging Integration
//
// Each Check Run includes a details URL that links to Cloud Logging with
// pre-configured filters for the specific PR and commit SHA, making it easy
// to view logs for that particular reconciliation run.
package statusmanager
