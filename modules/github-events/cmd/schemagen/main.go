/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/internal/schemagen"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/schemas"
)

var base = flag.String("base", "./modules/github-events/schemas", "base directory to write to")

func main() {
	flag.Parse()

	mustGenerate("pull_request.schema.json", schemas.Wrapper[schemas.PullRequestEvent]{})
	mustGenerate("workflow_run.schema.json", schemas.Wrapper[schemas.WorkflowRunEvent]{})
}

func mustGenerate[T any](path string, w schemas.Wrapper[T]) {
	if err := generate(path, w); err != nil {
		log.Fatalf("Failed to generate %T -> %s: %v", w, path, err)
	}
}

func generate[T any](fn string, w schemas.Wrapper[T]) error {
	return schemagen.Generate(filepath.Join(*base, fn), w)
}
