/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package workqueue

import (
	"log/slog"
	"time"
)

// LogAttrs returns a slice of attributes for logging purposes.
func (x *ProcessRequest) LogAttrs() []any {
	return []any{
		slog.String("key", x.Key),
		slog.Int64("priority", x.Priority),
		slog.Duration("delay", time.Duration(x.DelaySeconds)*time.Second),
	}
}
