/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package secrets_test

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/internal/secrets"
)

func ExampleLoadFromEnv() {
	ctx := context.Background()
	// LoadFromEnv reads all WEBHOOK_SECRET* environment variables.
	// With no such variables set, it returns nil.
	s := secrets.LoadFromEnv(ctx)
	fmt.Println(len(s))
	// Output: 0
}
