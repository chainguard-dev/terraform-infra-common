// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

output "emails" {
  description = "Sorted, lowercase emails of each group's members, including members of nested groups, keyed by group email. Suits the secret module's group_members input."
  value       = local.emails
}
