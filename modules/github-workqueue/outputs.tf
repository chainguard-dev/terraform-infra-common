/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "webhook" {
  depends_on = [module.shim]
  value = {
    name = local.shim_name
  }
}
