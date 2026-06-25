/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// newTestEvent creates a CloudEvent with the given type and ID for testing.
func newTestEvent(t *testing.T, eventType, eventID string, data []byte) cloudevents.Event {
	t.Helper()
	e := cloudevents.NewEvent()
	e.SetType(eventType)
	e.SetID(eventID)
	e.SetSource("test/source")
	e.SetTime(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	if err := e.SetData("application/json", data); err != nil {
		t.Fatalf("SetData() = %v", err)
	}
	return e
}

// Test_recordEvent_pathTraversalDotDotID verifies that a ce-id containing ".."
// does not write files outside the log directory. The event is acked (no
// error) but the traversal path must not be created.
//
// The ID is the last path component, placed under <logPath>/<type>/<date>/.
// To escape logPath entirely, the ID must traverse two levels up (past type
// and date), so we use "../../pwned".
func Test_recordEvent_pathTraversalDotDotID(t *testing.T) {
	logPath := t.TempDir()

	e := newTestEvent(t, "dev.chainguard.test", "../../pwned", []byte(`{}`))

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() returned unexpected error: %v", err)
	}

	// The traversal target must not exist outside the log directory.
	traversalTarget := filepath.Join(filepath.Dir(logPath), "pwned")
	if _, statErr := os.Stat(traversalTarget); statErr == nil {
		t.Errorf("path traversal succeeded: file %q was created outside log directory", traversalTarget)
	}
}

// Test_recordEvent_pathTraversalDotDotType verifies that a ce-type containing
// ".." does not write files outside the log directory.
func Test_recordEvent_pathTraversalDotDotType(t *testing.T) {
	logPath := t.TempDir()

	e := newTestEvent(t, "../pwned", "safe-id", []byte(`{}`))

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() returned unexpected error: %v", err)
	}

	// The traversal target must not exist outside the log directory.
	traversalTarget := filepath.Join(filepath.Dir(logPath), "pwned")
	if _, statErr := os.Stat(traversalTarget); statErr == nil {
		t.Errorf("path traversal succeeded: file %q was created outside log directory", traversalTarget)
	}
}

// Test_recordEvent_slashInID verifies that a ce-id containing a slash is
// encoded and the event is still recorded (no error, no traversal).
func Test_recordEvent_slashInID(t *testing.T) {
	logPath := t.TempDir()

	e := newTestEvent(t, "dev.chainguard.test", "foo/bar", []byte(`{}`))

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() = %v, want nil", err)
	}

	// The slash must have been encoded; no subdirectory "foo" should exist
	// directly under the type/date directory.
	typeDir := filepath.Join(logPath, "dev.chainguard.test", "2024-01-15")
	entries, err := os.ReadDir(typeDir)
	if err != nil {
		t.Fatalf("ReadDir(%q) = %v", typeDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == "foo" {
			t.Errorf("slash in ID created unexpected subdirectory %q (path traversal via slash)", entry.Name())
		}
	}
}

// Test_recordEvent_slashInType verifies that a ce-type containing a slash is
// encoded and the event is still recorded (no error, no traversal).
func Test_recordEvent_slashInType(t *testing.T) {
	logPath := t.TempDir()

	e := newTestEvent(t, "dev/chainguard/test", "safe-id", []byte(`{}`))

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() = %v, want nil", err)
	}

	// The slashes must have been encoded; no subdirectory "dev" should exist
	// directly under logPath.
	entries, err := os.ReadDir(logPath)
	if err != nil {
		t.Fatalf("ReadDir(%q) = %v", logPath, err)
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == "dev" {
			t.Errorf("slash in type created unexpected subdirectory %q (path traversal via slash)", entry.Name())
		}
	}
}

// Test_recordEvent_rfc3339ID verifies that a ce-id containing RFC3339
// characters (colons) is recorded successfully. RFC3339 timestamps are a
// common ID format; IDs like "run-abc2025-01-01T00:00:00Z" must be accepted.
func Test_recordEvent_rfc3339ID(t *testing.T) {
	logPath := t.TempDir()
	data := []byte(`{"key":"value"}`)

	e := newTestEvent(t, "dev.chainguard.infra.image.build.failure", "run-abc2025-01-01T00:00:00Z", data)

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() = %v, want nil (RFC3339 IDs must be accepted)", err)
	}

	// Verify a file was actually written under logPath.
	typeDir := filepath.Join(logPath, "dev.chainguard.infra.image.build.failure", "2024-01-15")
	entries, err := os.ReadDir(typeDir)
	if err != nil {
		t.Fatalf("ReadDir(%q) = %v, want nil", typeDir, err)
	}
	if len(entries) == 0 {
		t.Errorf("no file written for RFC3339 ID event")
	}
}

// Test_recordEvent_uidpID verifies that a ce-id containing slashes is recorded
// successfully (slashes are encoded, not rejected).
func Test_recordEvent_uidpID(t *testing.T) {
	logPath := t.TempDir()
	data := []byte(`{}`)

	// Simulate a UIDP-style ID: "org/project/resource".
	e := newTestEvent(t, "dev.chainguard.test", "org/project/resource", data)

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() = %v, want nil (UIDP IDs must be accepted)", err)
	}

	// Verify a file was actually written under logPath.
	typeDir := filepath.Join(logPath, "dev.chainguard.test", "2024-01-15")
	entries, err := os.ReadDir(typeDir)
	if err != nil {
		t.Fatalf("ReadDir(%q) = %v, want nil", typeDir, err)
	}
	if len(entries) == 0 {
		t.Errorf("no file written for UIDP ID event")
	}
}

// Test_recordEvent_valid verifies that a well-formed event is written to the
// correct path with the correct content.
func Test_recordEvent_valid(t *testing.T) {
	logPath := t.TempDir()
	data := []byte(`{"key":"value"}`)

	e := newTestEvent(t, "dev.chainguard.test", "abc-123_event.1", data)

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() = %v, want nil", err)
	}

	expectedPath := filepath.Join(logPath, "dev.chainguard.test", "2024-01-15", "abc-123_event.1")
	got, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) = %v, want nil", expectedPath, err)
	}
	if string(got) != string(data) {
		t.Errorf("file content = %q, want %q", got, data)
	}
}

// Test_recordEvent_emptyID verifies that an event with an empty ID is acked
// (returns nil) rather than retried.
func Test_recordEvent_emptyID(t *testing.T) {
	logPath := t.TempDir()

	e := newTestEvent(t, "dev.chainguard.test", "", []byte(`{}`))

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() = %v, want nil (empty ID must be acked, not retried)", err)
	}
}

// Test_recordEvent_emptyType verifies that an event with an empty type is
// acked (returns nil) rather than retried.
func Test_recordEvent_emptyType(t *testing.T) {
	logPath := t.TempDir()

	e := newTestEvent(t, "", "safe-id", []byte(`{}`))

	if err := recordEvent(t.Context(), logPath, e); err != nil {
		t.Fatalf("recordEvent() = %v, want nil (empty type must be acked, not retried)", err)
	}
}

// Test_sanitizePathComponent covers the encoding and rejection behavior of
// sanitizePathComponent.
func Test_sanitizePathComponent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "simple alphanumeric", input: "abc123", want: "abc123"},
		{name: "dots allowed in middle", input: "dev.chainguard.test", want: "dev.chainguard.test"},
		{name: "hyphens allowed", input: "my-event-type", want: "my-event-type"},
		{name: "underscores allowed", input: "my_event", want: "my_event"},
		{name: "mixed safe chars", input: "abc-123_event.1", want: "abc-123_event.1"},
		// Encoding cases — must succeed (not error) and produce a safe name.
		{name: "dot-dot traversal encoded", input: "..", want: "%2E."},
		{name: "slash encoded", input: "foo/bar", want: "foo%2Fbar"},
		{name: "colon encoded (RFC3339)", input: "run2025-01-01T00:00:00Z", want: "run2025-01-01T00%3A00%3A00Z"},
		{name: "leading dot encoded", input: ".hidden", want: "%2Ehidden"},
		{name: "single dot encoded", input: ".", want: "%2E"},
		{name: "trailing dot preserved", input: "abc.", want: "abc."},
		{name: "dot-dot-a encoded leading dot", input: "..a", want: "%2E.a"},
		{name: "a-dot-dot preserved", input: "a..", want: "a.."},
		{name: "null byte encoded", input: "foo\x00bar", want: "foo%00bar"},
		{name: "space encoded", input: "foo bar", want: "foo%20bar"},
		{name: "embedded newline encoded", input: "abc\nxyz", want: "abc%0Axyz"},
		// Error cases.
		{name: "empty string", input: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizePathComponent(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("sanitizePathComponent(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
				return
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("sanitizePathComponent(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
