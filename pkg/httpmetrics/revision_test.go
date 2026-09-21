/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics_test

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestServerRevisionHandler(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		revision string
		status   int
	}{
		{name: "success", revision: fmt.Sprintf("api-%x", rand.Uint64()), status: http.StatusOK},
		{name: "application error", revision: fmt.Sprintf("api-%x", rand.Uint64()), status: http.StatusInternalServerError},
		{name: "no revision", status: http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			handler := httpmetrics.ServerRevisionHandler(tc.revision, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if w != response {
					t.Error("response writer was wrapped; optional interfaces must remain available")
				}
				w.Header().Set("X-Application", "preserved")
				w.WriteHeader(tc.status)
				if _, err := io.WriteString(w, "body preserved"); err != nil {
					t.Fatal(err)
				}
			}))
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			request.Header.Set(httpmetrics.ServerRevisionHeader, "caller-supplied")
			handler.ServeHTTP(response, request)
			assertServerRevision(t, response.Header().Values(httpmetrics.ServerRevisionHeader), tc.revision)
			if got := response.Code; got != tc.status {
				t.Errorf("status: got = %d, want = %d", got, tc.status)
			}
			if got := response.Body.String(); got != "body preserved" {
				t.Errorf("body: got = %q, want = %q", got, "body preserved")
			}
			if got := response.Header().Get("X-Application"); got != "preserved" {
				t.Errorf("application header: got = %q, want = %q", got, "preserved")
			}
		})
	}
}

func TestServerRevisionAuthenticationOrder(t *testing.T) {
	t.Parallel()
	handler := httpmetrics.ServerRevisionHandler("api-00001-abc", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("unauthenticated request reached application")
	}))
	authenticated := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
	response := httptest.NewRecorder()
	authenticated.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	assertServerRevision(t, response.Header().Values(httpmetrics.ServerRevisionHeader), "")
	if got := response.Code; got != http.StatusUnauthorized {
		t.Errorf("status: got = %d, want = %d", got, http.StatusUnauthorized)
	}
}

func TestServerRevisionGRPC(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		revision     string
		code         codes.Code
		rejectBefore bool
		headersSent  bool
	}{
		{name: "success", revision: fmt.Sprintf("api-%x", rand.Uint64())},
		{name: "no revision"},
		{name: "application error", revision: fmt.Sprintf("api-%x", rand.Uint64()), code: codes.PermissionDenied},
		{name: "authentication error", revision: fmt.Sprintf("api-%x", rand.Uint64()), code: codes.Unauthenticated, rejectBefore: true},
		{name: "headers already sent", revision: fmt.Sprintf("api-%x", rand.Uint64()), headersSent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			unary := []grpc.UnaryServerInterceptor{httpmetrics.ServerRevisionUnaryInterceptor(tc.revision)}
			stream := []grpc.StreamServerInterceptor{httpmetrics.ServerRevisionStreamInterceptor(tc.revision)}
			if tc.headersSent {
				unary = append([]grpc.UnaryServerInterceptor{func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
					if err := grpc.SendHeader(ctx, metadata.Pairs("x-earlier-header", "sent")); err != nil {
						return nil, err
					}
					return handler(ctx, req)
				}}, unary...)
				stream = append([]grpc.StreamServerInterceptor{func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
					if err := stream.SendHeader(metadata.Pairs("x-earlier-header", "sent")); err != nil {
						return err
					}
					return handler(srv, stream)
				}}, stream...)
			}
			if tc.code != codes.OK {
				rejectUnary := func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error) {
					return nil, status.Error(tc.code, "request rejected")
				}
				rejectStream := func(any, grpc.ServerStream, *grpc.StreamServerInfo, grpc.StreamHandler) error {
					return status.Error(tc.code, "request rejected")
				}
				if tc.rejectBefore {
					unary = append([]grpc.UnaryServerInterceptor{rejectUnary}, unary...)
					stream = append([]grpc.StreamServerInterceptor{rejectStream}, stream...)
				} else {
					unary = append(unary, rejectUnary)
					stream = append(stream, rejectStream)
				}
			}
			client := revisionHealthClient(t, grpc.ChainUnaryInterceptor(unary...), grpc.ChainStreamInterceptor(stream...))
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			// A caller cannot choose the revision reported by the server.
			ctx = metadata.AppendToOutgoingContext(ctx, httpmetrics.ServerRevisionHeader, "caller-supplied")
			wantRevision := tc.revision
			if tc.rejectBefore || tc.headersSent {
				wantRevision = ""
			}
			var headers metadata.MD
			response, err := client.Check(ctx, &healthpb.HealthCheckRequest{}, grpc.Header(&headers))
			if got := status.Code(err); got != tc.code {
				t.Fatalf("unary code: got = %s, want = %s (%v)", got, tc.code, err)
			}
			if tc.code == codes.OK && response.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				t.Errorf("unary response: got = %s, want = SERVING", response.GetStatus())
			}
			assertServerRevision(t, headers.Get(httpmetrics.ServerRevisionHeader), wantRevision)

			watch, err := client.Watch(ctx, &healthpb.HealthCheckRequest{})
			if err != nil {
				t.Fatal(err)
			}
			response, err = watch.Recv()
			if got := status.Code(err); got != tc.code {
				t.Fatalf("stream code: got = %s, want = %s (%v)", got, tc.code, err)
			}
			if tc.code == codes.OK && response.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				t.Errorf("stream response: got = %s, want = SERVING", response.GetStatus())
			}
			headers, err = watch.Header()
			if err != nil {
				t.Fatal(err)
			}
			assertServerRevision(t, headers.Get(httpmetrics.ServerRevisionHeader), wantRevision)
		})
	}
}

func TestServerRevisionConcurrentUnary(t *testing.T) {
	t.Parallel()
	revision := fmt.Sprintf("api-%x", rand.Uint64())
	client := revisionHealthClient(t, grpc.ChainUnaryInterceptor(httpmetrics.ServerRevisionUnaryInterceptor(revision)))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var group errgroup.Group
	for range 20 {
		group.Go(func() error {
			var headers metadata.MD
			if _, err := client.Check(ctx, &healthpb.HealthCheckRequest{}, grpc.Header(&headers)); err != nil {
				return err
			}
			if got := headers.Get(httpmetrics.ServerRevisionHeader); !slices.Equal(got, []string{revision}) {
				return fmt.Errorf("revision: got = %q, want = %q", got, []string{revision})
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		t.Fatal(err)
	}
}

func revisionHealthClient(t *testing.T, opts ...grpc.ServerOption) healthpb.HealthClient {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer(opts...)
	healthpb.RegisterHealthServer(server, health.NewServer())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := server.Serve(listener); err != nil {
			t.Errorf("serving gRPC: %v", err)
		}
	}()
	t.Cleanup(func() {
		server.Stop()
		<-done
	})
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return healthpb.NewHealthClient(conn)
}

func assertServerRevision(t *testing.T, got []string, revision string) {
	t.Helper()
	var want []string
	if revision != "" {
		want = []string{revision}
	}
	if !slices.Equal(got, want) {
		t.Errorf("revision: got = %q, want = %q", got, want)
	}
}
