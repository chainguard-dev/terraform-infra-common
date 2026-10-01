/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package secrets

import (
	"testing"

	"github.com/chainguard-dev/clog/slogtest"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/internal/trampoline"
	"github.com/google/go-cmp/cmp"
)

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		env         [][2]string
		wantUnbound [][]byte
		wantBound   []trampoline.BoundSecret
	}{{
		name:        "secrets without bindings are unbound",
		env:         [][2]string{{"WEBHOOK_SECRET", "foo"}, {"WEBHOOK_SECRET_2", "bar"}},
		wantUnbound: [][]byte{[]byte("foo"), []byte("bar")},
	}, {
		name: "binding attaches hook IDs to the secret with the same name",
		env: [][2]string{
			{"WEBHOOK_SECRET", "foo"},
			{"WEBHOOK_SECRET_APP", "bar"},
			{"BOUND_WEBHOOK_IDS_APP", "123,456"},
		},
		wantUnbound: [][]byte{[]byte("foo")},
		wantBound:   []trampoline.BoundSecret{{Secret: []byte("bar"), HookIDs: []string{"123", "456"}}},
	}, {
		name: "binding without a matching secret is never loaded as a secret",
		env:  [][2]string{{"BOUND_WEBHOOK_IDS_APP", "123"}},
	}, {
		name:        "binding for another name leaves the secret unbound",
		env:         [][2]string{{"WEBHOOK_SECRET_APP", "bar"}, {"BOUND_WEBHOOK_IDS_OTHER", "123"}},
		wantUnbound: [][]byte{[]byte("bar")},
	}, {
		name:        "primary secret ignores a binding with an empty name",
		env:         [][2]string{{"WEBHOOK_SECRET", "foo"}, {"BOUND_WEBHOOK_IDS_", "123"}},
		wantUnbound: [][]byte{[]byte("foo")},
	}, {
		name:      "binding trims spaces and skips empty entries",
		env:       [][2]string{{"WEBHOOK_SECRET_APP", "bar"}, {"BOUND_WEBHOOK_IDS_APP", " 123, ,456,"}},
		wantBound: []trampoline.BoundSecret{{Secret: []byte("bar"), HookIDs: []string{"123", "456"}}},
	}, {
		name:      "empty binding stays bound to no hooks",
		env:       [][2]string{{"WEBHOOK_SECRET_APP", "bar"}, {"BOUND_WEBHOOK_IDS_APP", ""}},
		wantBound: []trampoline.BoundSecret{{Secret: []byte("bar")}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, kv := range tt.env {
				t.Setenv(kv[0], kv[1])
			}

			gotUnbound, gotBound := LoadFromEnv(slogtest.Context(t))
			if diff := cmp.Diff(tt.wantUnbound, gotUnbound); diff != "" {
				t.Errorf("LoadFromEnv() unbound mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantBound, gotBound); diff != "" {
				t.Errorf("LoadFromEnv() bound mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
