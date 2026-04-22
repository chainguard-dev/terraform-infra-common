/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package secrets provides utilities for loading webhook secrets from the
// environment.
//
// Use [LoadFromEnv] to collect all environment variables whose names begin
// with "WEBHOOK_SECRET" as webhook signing secrets.
package secrets
