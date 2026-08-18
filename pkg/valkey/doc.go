/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package valkey connects to Memorystore for Valkey instances created by the
// terraform valkey module. It applies the same opinions the module hardcodes:
// IAM_AUTH as the workload identity (a fresh token per reconnect), TLS pinned
// to the managed server CA, and the PSC connect endpoint. Resolve the
// instance's full resource name once at boot, then dial clients from the
// resolved Endpoint.
package valkey
