/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Generate the proto definitions
//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative workqueue.proto

// Package workqueue contains an interface for a simple key
// workqueue abstraction.
package workqueue
