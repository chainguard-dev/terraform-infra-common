/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package inmem

import (
	"testing"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/conformance"
)

func TestWorkQueue(t *testing.T) {
	conformance.TestSemantics(t, NewWorkQueue)

	conformance.TestConcurrency(t, NewWorkQueue)
}
