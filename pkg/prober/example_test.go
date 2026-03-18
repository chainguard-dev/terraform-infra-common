/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package prober_test

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/pkg/prober"
)

func ExampleFunc() {
	// Func adapts a plain function into a prober.Interface.
	p := prober.Func(func(_ context.Context) error {
		fmt.Println("probing...")
		return nil
	})
	_ = p
}
