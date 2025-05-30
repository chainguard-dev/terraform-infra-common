/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package secrets

import (
	"context"
	"os"
	"strings"

	"github.com/chainguard-dev/clog"
)

func LoadFromEnv(ctx context.Context) [][]byte {
	var secrets [][]byte
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}

		if strings.HasPrefix(k, "WEBHOOK_SECRET") {
			clog.InfoContextf(ctx, "loading secret: %q", k)
			secrets = append(secrets, []byte(v))
		}
	}
	return secrets
}
