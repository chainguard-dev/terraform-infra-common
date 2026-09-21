/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ServerRevisionHeader identifies the revision that served a response. It is
// shared by HTTP response headers and gRPC response metadata. The value is
// diagnostic information, not an authentication or authorization credential.
const ServerRevisionHeader = "x-chainguard-revision"

// ServerRevisionHandler adds revision to HTTP responses before calling handler,
// including responses with error statuses. Pass the service's K_REVISION value
// at startup; an empty revision leaves responses unchanged. It passes the
// original ResponseWriter through, preserving support for streaming and hijacking.
// Place it inside authentication middleware to report only on authenticated
// requests. The handler is safe for concurrent use if handler is.
func ServerRevisionHandler(revision string, handler http.Handler) http.Handler {
	if revision == "" {
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(ServerRevisionHeader, revision)
		handler.ServeHTTP(w, r)
	})
}

// ServerRevisionUnaryInterceptor identifies the serving revision in unary gRPC
// response headers, including application errors. Pass the service's K_REVISION
// value at startup. An empty revision emits no metadata. Install it once, after
// authentication, and reserve ServerRevisionHeader for this interceptor. It is
// safe for concurrent use. If headers cannot be added (for example, an earlier
// interceptor already sent them), the application handler still runs unchanged.
func ServerRevisionUnaryInterceptor(revision string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if revision != "" {
			// Diagnostic metadata must not change the application's outcome.
			_ = grpc.SetHeader(ctx, metadata.Pairs(ServerRevisionHeader, revision))
		}
		return handler(ctx, req)
	}
}

// ServerRevisionStreamInterceptor identifies the serving revision in streaming
// gRPC response headers, including application errors. Metadata is queued for
// the stream's normal header send. Pass the service's K_REVISION value at
// startup. An empty revision emits no metadata. Install it once, after
// authentication, and reserve ServerRevisionHeader for this interceptor. It is
// safe for concurrent use. If headers cannot be added (for example, an earlier
// interceptor already sent them), the application handler still runs unchanged.
func ServerRevisionStreamInterceptor(revision string) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if revision != "" {
			// Diagnostic metadata must not change the application's outcome.
			_ = stream.SetHeader(metadata.Pairs(ServerRevisionHeader, revision))
		}
		return handler(srv, stream)
	}
}
