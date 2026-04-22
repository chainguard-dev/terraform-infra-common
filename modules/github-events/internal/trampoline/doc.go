/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package trampoline implements an HTTP server that receives GitHub webhook
// events, validates their signatures, and forwards them as CloudEvents to a
// broker ingress.
//
// Use [NewServer] to create a [Server] configured with a CloudEvents client
// and webhook secrets. The server implements [http.Handler] and can be
// registered with any HTTP mux.
package trampoline
