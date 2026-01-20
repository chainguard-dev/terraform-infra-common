/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "receiver" {
  depends_on  = [module.hyperqueue-service]
  description = "The hyperqueue router service (clients queue work here)"
  value = {
    name = "${var.name}-hq"
  }
}
