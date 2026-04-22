/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package schemas_test

import (
	"fmt"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/schemas"
)

func ExampleWrapper() {
	w := schemas.Wrapper[schemas.PullRequest]{
		When: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	fmt.Println(w.When.Year())
	// Output: 2025
}
