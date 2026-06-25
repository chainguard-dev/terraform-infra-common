/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/sethvargo/go-envconfig"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	"github.com/chainguard-dev/terraform-infra-common/pkg/profiler"
)

// sanitizePathComponent encodes s into a string that is safe to use as a
// single filesystem path component. Characters outside [A-Za-z0-9._-] are
// percent-encoded so that legitimate CloudEvent IDs (e.g. RFC3339 timestamps
// containing ':') and UIDPs containing '/' are preserved without enabling
// path traversal. Dots in non-leading positions are kept as-is; a leading dot
// is percent-encoded to prevent hidden files and ".."-style traversal.
//
// The function iterates over bytes (not runes) so that invalid UTF-8 sequences
// are encoded faithfully, making the mapping injective: two distinct byte
// strings always produce two distinct encoded strings.
func sanitizePathComponent(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("path component must not be empty")
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := range len(s) {
		c := s[i]
		switch {
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-':
			b.WriteByte(c)
		case c == '.':
			if i == 0 {
				// Encode a leading dot to prevent ".." traversal and hidden files.
				fmt.Fprintf(&b, "%%%02X", '.')
			} else {
				b.WriteByte(c)
			}
		default:
			// Percent-encode everything else (slashes, colons, spaces, null
			// bytes, high bytes, …) byte-by-byte.
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String(), nil
}

// recordEvent writes the event data to a file under logPath, using the event
// type, date, and ID as path components. Unsafe characters in the type and ID
// are percent-encoded so that legitimate events with RFC3339 or UIDP IDs are
// still recorded. Permanently invalid events (empty type/ID) are logged and
// acked rather than retried.
func recordEvent(ctx context.Context, logPath string, event cloudevents.Event) error {
	safeType, err := sanitizePathComponent(event.Type())
	if err != nil {
		// Permanent failure — log and ack so Pub/Sub does not retry.
		clog.WarnContextf(ctx, "dropping event with invalid type %q: %v", event.Type(), err)
		return nil
	}
	safeID, err := sanitizePathComponent(event.ID())
	if err != nil {
		// Permanent failure — log and ack so Pub/Sub does not retry.
		clog.WarnContextf(ctx, "dropping event with invalid ID %q: %v", event.ID(), err)
		return nil
	}

	dir := filepath.Join(logPath, safeType, event.Time().Format("2006-01-02"))
	filename := filepath.Join(dir, safeID)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filename, event.Data(), 0600); err != nil {
		clog.WarnContextf(ctx, "failed to write file %s; %v", filename, err)
		if err := os.RemoveAll(filename); err != nil {
			clog.WarnContextf(ctx, "failed to remove failed write file: %s; %v", filename, err)
		}
		return err
	}
	return nil
}

var env = envconfig.MustProcess(context.Background(), &struct {
	Port    int    `env:"PORT,default=8080"`
	LogPath string `env:"LOG_PATH"`
}{})

func main() {
	if env.LogPath == "" {
		clog.Fatalf("LOG_PATH environment variable is required")
	}

	profiler.SetupProfiler()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	c, err := mce.NewClientHTTP("ce-recorder", cloudevents.WithPort(env.Port))
	if err != nil {
		clog.Fatalf("failed to create event client, %v", err)
	}
	if err := c.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) error {
		return recordEvent(ctx, env.LogPath, event)
	}); err != nil {
		clog.Fatalf("failed to start event receiver, %v", err)
	}
}
