# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {}

variables {
  name           = "fixture"
  project_id     = "fixture-project"
  zone           = "us-central1-a"
  network        = "fixture-network"
  subnetwork     = "fixture-subnetwork"
  dev_principals = []
  team           = "fixture"
}

run "pre_startup_script_defaults_to_nothing" {
  command = plan
  assert {
    condition     = startswith(google_compute_instance.bastion.metadata_startup_script, "#!/usr/bin/env bash\nset -euo pipefail\n\napt-get update -y\n")
    error_message = "With no pre_startup_script the startup script must be unchanged, so existing bastions are not re-created"
  }
}

run "pre_startup_script_runs_before_the_package_installs" {
  command = plan
  variables {
    pre_startup_script = "echo pre"
    startup_script     = "echo caller"
  }
  assert {
    condition     = startswith(google_compute_instance.bastion.metadata_startup_script, "#!/usr/bin/env bash\nset -euo pipefail\n\necho pre\napt-get update -y\n")
    error_message = "pre_startup_script must run right after the shell options, before apt"
  }
  assert {
    condition     = can(regex("augenrules --load\\n+echo caller\\n", google_compute_instance.bastion.metadata_startup_script))
    error_message = "startup_script keeps its place after the auditd install"
  }
}
