/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package tracesplit_test

import (
	"context"

	"github.com/chainguard-dev/terraform-infra-common/pkg/tracesplit"
)

func ExampleSplit() {
	// A build emits enough spans to warrant its own trace.
	ctx := context.Background()
	ctx, span := tracesplit.Split(ctx, "apko-build", tracesplit.WithRelatedTraceID())
	defer span.End()
	_ = ctx // run the unit of work with ctx
}
