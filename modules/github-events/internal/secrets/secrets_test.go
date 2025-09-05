/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package secrets

import (
	"testing"

	"github.com/chainguard-dev/clog/slogtest"
	"github.com/google/go-cmp/cmp"
)

func TestLoadFromEnv(t *testing.T) {
	// Set up the environment variables for testing.
	t.Setenv("WEBHOOK_SECRET", "foo")
	t.Setenv("WEBHOOK_SECRET_2", "bar")

	ctx := slogtest.Context(t)
	got := LoadFromEnv(ctx)

	want := [][]byte{[]byte("foo"), []byte("bar")}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("LoadFromEnv() mismatch (-want +got):\n%s", diff)
	}
}
