/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/
package quit

import (
	"context"
)

var (
	_ = context.AfterFunc(context.TODO(), Sidecar(12345))
	_ = context.AfterFunc(context.TODO(), OtelSidecar())
)
