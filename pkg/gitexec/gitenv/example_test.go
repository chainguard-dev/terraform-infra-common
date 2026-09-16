/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitenv_test

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/pkg/gitexec/gitenv"
)

func ExampleEnviron() {
	env, err := gitenv.Environ(gitenv.Options{
		Remote: "https://github.com/example/repo",
	})
	if err != nil {
		panic(err)
	}
	// env contains the sanitised environment for a git subprocess.
	_ = env
}

func ExampleCheckVersion() {
	ctx := context.Background()
	if err := gitenv.CheckVersion(ctx); err != nil {
		fmt.Println("git version check failed:", err)
		return
	}
	fmt.Println("git version ok")
}
