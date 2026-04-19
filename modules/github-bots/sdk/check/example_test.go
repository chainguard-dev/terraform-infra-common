/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package check_test

import (
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk/check"
)

func ExampleNewBuilder() {
	b := check.NewBuilder("my-check", "abc123")
	b.Status = check.StatusInProgress
	b.Writef("Running checks...")

	cr := b.CheckRunCreate()
	fmt.Println(cr.Name)
	fmt.Println(cr.GetStatus())
	// Output:
	// my-check
	// in_progress
}

func ExampleBuilder_Writef() {
	b := check.NewBuilder("my-check", "abc123")
	b.Writef("step %d: %s", 1, "passed")
	b.Writef("step %d: %s", 2, "passed")

	cr := b.CheckRunCreate()
	fmt.Print(cr.GetOutput().GetText())
	// Output:
	// step 1: passed
	// step 2: passed
}

func ExampleBuilder_CheckRunCreate() {
	b := check.NewBuilder("my-check", "abc123")
	b.Conclusion = check.ConclusionSuccess
	b.Summary = "All checks passed"

	cr := b.CheckRunCreate()
	fmt.Println(cr.GetStatus())
	fmt.Println(cr.GetConclusion())
	// Output:
	// completed
	// success
}

func ExampleBuilder_CheckRunUpdate() {
	b := check.NewBuilder("my-check", "abc123")
	b.Conclusion = check.ConclusionFailure

	u := b.CheckRunUpdate()
	fmt.Println(u.GetConclusion())
	// Output: failure
}
