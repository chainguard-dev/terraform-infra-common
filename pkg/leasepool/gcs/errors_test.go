/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/internal/engine"
)

// Exercise the real clients' upload error paths, including buffering and Close.
// These endpoints inject errors only; they do not model successful GCS writes.
func TestWriteErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		httpCode int
		grpcCode codes.Code
		gen      int64
		conflict bool
	}{
		{"create_conflict", http.StatusPreconditionFailed, codes.FailedPrecondition, 0, true},
		{"replace_conflict", http.StatusPreconditionFailed, codes.FailedPrecondition, 1, true},
		{"missing_generation", http.StatusNotFound, codes.NotFound, 1, true},
		{"missing_bucket", http.StatusNotFound, codes.NotFound, 0, false},
		{"permission_denied", http.StatusForbidden, codes.PermissionDenied, 1, false},
		{"unavailable", http.StatusServiceUnavailable, codes.Unavailable, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, transport := range []string{"http", "grpc"} {
				t.Run(transport, func(t *testing.T) {
					var client *storage.Client
					if transport == "http" {
						server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(tc.httpCode)
							fmt.Fprintf(w, `{"error":{"code":%d,"message":"injected failure"}}`, tc.httpCode)
						}))
						t.Cleanup(server.Close)
						var err error
						client, err = storage.NewClient(t.Context(), option.WithEndpoint(server.URL), option.WithHTTPClient(server.Client()), option.WithoutAuthentication())
						if err != nil {
							t.Fatal(err)
						}
					} else {
						listener := bufconn.Listen(1 << 20)
						t.Cleanup(func() { _ = listener.Close() })
						server := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
							method, _ := grpc.MethodFromServerStream(stream)
							if method != "/google.storage.v2.Storage/BidiWriteObject" {
								t.Errorf("unexpected RPC: %s", method)
							}
							return status.Error(tc.grpcCode, "injected failure")
						}))
						done := make(chan struct{})
						go func() {
							defer close(done)
							if err := server.Serve(listener); err != nil {
								t.Errorf("serve: %v", err)
							}
						}()
						t.Cleanup(func() { server.Stop(); <-done })
						conn, err := grpc.NewClient("passthrough:///lease-test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
							return listener.DialContext(ctx)
						}))
						if err != nil {
							t.Fatal(err)
						}
						t.Cleanup(func() { _ = conn.Close() })
						client, err = storage.NewGRPCClient(t.Context(), option.WithGRPCConn(conn), option.WithoutAuthentication())
						if err != nil {
							t.Fatal(err)
						}
					}
					t.Cleanup(func() { _ = client.Close() })
					// Classify a single failed RPC; retry behavior has its own regression.
					s := store{bucket: client.Bucket("leases").Retryer(storage.WithPolicy(storage.RetryNever))}
					_, err := s.Write(t.Context(), "name", engine.Record{Version: 1}, tc.gen)
					switch {
					case tc.conflict:
						if !errors.Is(err, engine.ErrConflict) {
							t.Fatalf("got %v, want conflict", err)
						}
					case transport == "http":
						if ge, ok := errors.AsType[*googleapi.Error](err); !ok || ge.Code != tc.httpCode {
							t.Fatalf("got %v, want HTTP %d", err, tc.httpCode)
						}
					case status.Code(err) != tc.grpcCode:
						t.Fatalf("got %v, want gRPC %v", err, tc.grpcCode)
					}
				})
			}
		})
	}
}

// Keep the real upload client and retry policy. The endpoint injects one transient
// failure and asserts every attempt carries the identical CAS precondition.
func TestWriteRetriesTransientError(t *testing.T) {
	for _, gen := range []int64{0, 1} {
		t.Run(strconv.FormatInt(gen, 10), func(t *testing.T) {
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("ifGenerationMatch"); got != strconv.FormatInt(gen, 10) {
					t.Errorf("generation precondition = %q, want %d", got, gen)
				}
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set("Content-Type", "application/json")
				if attempts.Add(1) == 1 {
					w.WriteHeader(http.StatusServiceUnavailable)
					fmt.Fprint(w, `{"error":{"code":503,"message":"transient failure"}}`)
					return
				}
				fmt.Fprint(w, `{"generation":"2"}`)
			}))
			defer server.Close()
			client, err := storage.NewClient(t.Context(), option.WithEndpoint(server.URL), option.WithHTTPClient(server.Client()), option.WithoutAuthentication())
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			s := store{bucket: client.Bucket("leases")}
			got, err := s.Write(t.Context(), "name", engine.Record{Version: 1}, gen)
			if err != nil || got != 2 {
				t.Fatalf("Write = %d, %v, want generation 2", got, err)
			}
			if got := attempts.Load(); got != 2 {
				t.Errorf("attempts = %d, want 2", got)
			}
		})
	}
}

func TestPrefixLength(t *testing.T) {
	client, err := storage.NewClient(t.Context(), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, tc := range []struct {
		name, prefix string
		invalid      bool
	}{
		{name: "empty"}, {name: "maximum", prefix: strings.Repeat("p", 255)},
		{name: "too long", prefix: strings.Repeat("p", 256), invalid: true},
		{name: "UTF-8 bytes", prefix: strings.Repeat("é", 128), invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(client.Bucket("leases"), tc.prefix, leasepool.Config{})
			if (err != nil) != tc.invalid {
				t.Errorf("New = %v, want invalid=%t", err, tc.invalid)
			}
		})
	}
}

// Exercise allocation through the real buffered HTTP upload client. Keep this
// serial so other test writes do not contaminate the process-wide counter.
func TestWriteAllocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"generation":"2"}`)
	}))
	defer server.Close()
	client, err := storage.NewClient(t.Context(), option.WithEndpoint(server.URL), option.WithHTTPClient(server.Client()), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	s := store{bucket: client.Bucket("leases")}
	write := func() {
		t.Helper()
		if _, err := s.Write(t.Context(), "name", engine.Record{Version: 1}, 1); err != nil {
			t.Fatal(err)
		}
	}
	write() // Exclude connection and client initialization.
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	const writes = 10
	for range writes {
		write()
	}
	runtime.ReadMemStats(&after)
	if got := (after.TotalAlloc - before.TotalAlloc) / writes; got >= 1<<20 {
		t.Fatalf("allocated bytes per write = %d, want less than 1 MiB", got)
	}
}
