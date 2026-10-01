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
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/internal/trampoline"
)

const (
	secretPrefix = "WEBHOOK_SECRET"
	// boundPrefix must not start with secretPrefix, or bindings would load as secrets.
	boundPrefix = "BOUND_WEBHOOK_IDS_"
)

// LoadFromEnv loads every environment variable whose name starts with
// WEBHOOK_SECRET as a webhook secret. WEBHOOK_SECRET_<NAME> is bound to the
// comma-separated hook IDs in BOUND_WEBHOOK_IDS_<NAME> when that variable is
// set; every other secret is unbound.
func LoadFromEnv(ctx context.Context) (unbound [][]byte, bound []trampoline.BoundSecret) {
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if !ok || !strings.HasPrefix(k, secretPrefix) {
			continue
		}
		clog.InfoContextf(ctx, "loading secret: %q", k)

		name, ok := strings.CutPrefix(k, secretPrefix+"_")
		ids, isBound := os.LookupEnv(boundPrefix + name)
		if !ok || name == "" || !isBound {
			unbound = append(unbound, []byte(v))
			continue
		}

		var hookIDs []string
		for id := range strings.SplitSeq(ids, ",") {
			if id = strings.TrimSpace(id); id != "" {
				hookIDs = append(hookIDs, id)
			}
		}
		if len(hookIDs) == 0 {
			clog.WarnContextf(ctx, "secret %q is bound to no hook IDs and will reject every delivery it signs", k)
		}
		clog.InfoContextf(ctx, "binding secret %q to hook IDs %q", k, hookIDs)
		bound = append(bound, trampoline.BoundSecret{Secret: []byte(v), HookIDs: hookIDs})
	}
	return unbound, bound
}
