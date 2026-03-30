# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands
- Run tests: `go test -race -timeout=5m ./...`
- Run single test: `go test -race -timeout=5m -run TestName ./path/to/package`
- Generate code: `go generate ./...`
- Validate Terraform: `terraform init && terraform validate`
- Format Terraform: `terraform fmt`
- Update Terraform docs: `bash hack/update-docs.sh`

## Code Style Guidelines
- **Go Formatting**: Use `gofmt -s` and `goimports`
- **License**: Files require Apache 2.0 license header
- **Headers**: Files begin with `Copyright YYYY Chainguard, Inc.` and `SPDX-License-Identifier: Apache-2.0`
- **Imports**: Group standard, external, and internal imports with blank lines between
- **Whitespace**: No trailing whitespace at end of lines, ensure newline at end of file
- **Error Handling**: Include context in errors, use structured logging
- **Naming**: Use idiomatic Go naming (CamelCase for exported, camelCase for private)
- **Comments**: Document all exported functions with comments
- **Testing**: Write unit tests with descriptive function names (Test_functionName_scenario)
- **Environment**: Process environment variables with envconfig
- **Project Structure**: Go code in /cmd and /pkg, Terraform modules in /modules

## Module Catalog

Before creating new modules or writing Terraform that provisions Cloud Run services, event routing, networking, databases, or observability, consult [MODULES.md](./MODULES.md) for existing modules. Prefer composing existing modules over writing raw GCP resources.

## Related Catalogs

- [terraform/MODULES.md](../../terraform/MODULES.md) — AWS, GCP, Kubernetes, and Chainguard private infrastructure modules
- [public/terraform-infra-reconcilers/MODULES.md](../terraform-infra-reconcilers/MODULES.md) — Workqueue-based reconciliation systems (DriftlessAF)
