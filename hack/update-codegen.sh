#!/usr/bin/env bash

# Copyright 2025 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# Regenerate protobuf files and JSON schemas.
go generate ./...
