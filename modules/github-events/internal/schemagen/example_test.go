/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package schemagen_test

import (
	"os"
	"path/filepath"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/internal/schemagen"
)

func ExampleGenerate() {
	type MyEvent struct {
		Action string `json:"action"`
	}

	path := filepath.Join(os.TempDir(), "schema.json")
	if err := schemagen.Generate(path, MyEvent{}); err != nil {
		panic(err)
	}
	// Output:
}
