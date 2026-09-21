# HTTP metrics and serving revisions

This package provides HTTP metrics, tracing, and optional serving-revision
metadata for Cloud Run services.

## Serving-revision metadata

The revision helpers attach `x-chainguard-revision` to HTTP response headers or
gRPC response metadata. Pass the process's `K_REVISION` value when constructing
the handler or server. When the value is empty, no revision metadata is added;
local development does not report a fabricated `unknown` revision.

HTTP services can compose the helper with their existing instrumentation:

```go
handler := httpmetrics.Handler("api",
    httpmetrics.ServerRevisionHandler(os.Getenv("K_REVISION"), apiHandler))
```

For native gRPC servers, register the unary and streaming interceptors:

```go
revision := os.Getenv("K_REVISION")
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        authenticateUnary,
        httpmetrics.ServerRevisionUnaryInterceptor(revision),
    ),
    grpc.ChainStreamInterceptor(
        authenticateStream,
        httpmetrics.ServerRevisionStreamInterceptor(revision),
    ),
)
```

Authentication should run first if only authenticated responses should expose
the revision. For HTTP, place authentication middleware outside
`ServerRevisionHandler`. Install each helper once and reserve
`ServerRevisionHeader` for it. The helpers are opt-in: existing metrics handlers,
services, and clients retain their current behavior until explicitly wired up.

The header accompanies application errors as well as successful responses. A
request rejected before reaching the helper, such as by Cloud Run IAM or an
earlier authentication interceptor, may have no revision. If an earlier gRPC
interceptor has already sent headers, the revision cannot be added; the
application still runs and keeps its normal result. Revision metadata is best
effort and never changes a gRPC application's result. Streaming gRPC queues the
metadata for the normal header send, usually the first response or final status;
it does not send an early response just to expose the revision. HTTP retains the
original response writer and its streaming and hijacking capabilities.

HTTP clients read `response.Header.Values(httpmetrics.ServerRevisionHeader)`.
Unary gRPC clients pass `grpc.Header(&headers)` and read
`headers.Get(httpmetrics.ServerRevisionHeader)`; streaming clients use
`stream.Header()`. A probe attributing a response to a revision should require
exactly one non-empty value. Missing or multiple values are inconclusive. Treat
the value as diagnostic data from the responding service, not as proof of
identity or a credential.
