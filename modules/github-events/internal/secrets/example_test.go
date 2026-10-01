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
	// LoadFromEnv reads all WEBHOOK_SECRET* environment variables and binds
	// WEBHOOK_SECRET_<NAME> to the hook IDs in BOUND_WEBHOOK_IDS_<NAME>.
	// With no such variables set, it returns nil for both.
	unbound, bound := secrets.LoadFromEnv(ctx)
	fmt.Println(len(unbound), len(bound))
	// Output: 0 0
}
