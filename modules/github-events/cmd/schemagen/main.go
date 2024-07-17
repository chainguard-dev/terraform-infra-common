/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

//go:generate go run ./

package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/internal/schemagen"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/schemas"
)

var base = flag.String("base", "./../../schemas", "base directory to write to")

func main() {
	flag.Parse()

	mustGenerate("pull_request.schema.json", schemas.Wrapper[schemas.PullRequestEvent]{})
	mustGenerate("workflow_run.schema.json", schemas.Wrapper[schemas.WorkflowRunEvent]{})
	mustGenerate("issue_comment.schema.json", schemas.Wrapper[schemas.IssueCommentEvent]{})
	mustGenerate("issues.schema.json", schemas.Wrapper[schemas.IssueEvent]{})
	mustGenerate("push.schema.json", schemas.Wrapper[schemas.PushEvent]{})
}

func mustGenerate[T any](path string, w schemas.Wrapper[T]) {
	if err := generate(path, w); err != nil {
		log.Fatalf("Failed to generate %T -> %s: %v", w, path, err)
	}
}

func generate[T any](fn string, w schemas.Wrapper[T]) error {
	return schemagen.Generate(filepath.Join(*base, fn), w)
}
