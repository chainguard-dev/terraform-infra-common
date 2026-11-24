# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

variables {
  prefix = "fixture-service"
  suffix = "-abc123"
}

run "under_limit_keeps_both_parts" {
  command = plan

  variables {
    limit = 63
  }

  assert {
    condition     = output.result == "fixture-service-abc123"
    error_message = "a result under limit must be the untouched concatenation."
  }
}

run "exact_fit_is_not_truncated" {
  command = plan

  variables {
    limit = 22
  }

  assert {
    condition     = output.result == "fixture-service-abc123"
    error_message = "a result exactly at limit must not lose characters."
  }
}

run "one_over_limit_drops_last_prefix_character" {
  command = plan

  variables {
    limit = 21
  }

  assert {
    condition     = output.result == "fixture-servic-abc123"
    error_message = "truncation must shorten the prefix and keep the suffix whole."
  }
}

run "long_prefix_truncates_to_limit" {
  command = plan

  variables {
    prefix = replace(format("%100s", ""), " ", "a")
    limit  = 63
  }

  assert {
    condition     = output.result == "${replace(format("%56s", ""), " ", "a")}-abc123"
    error_message = "a long prefix must be cut so the result is exactly limit characters ending in the suffix."
  }
}

run "limit_equal_to_suffix_keeps_only_suffix" {
  command = plan

  variables {
    limit = 7
  }

  assert {
    condition     = output.result == "-abc123"
    error_message = "a limit equal to the suffix length must drop the whole prefix."
  }
}

run "limit_below_suffix_length_is_rejected" {
  command = plan

  variables {
    limit = 6
  }

  expect_failures = [var.limit]
}

run "empty_suffix_truncates_prefix_alone" {
  command = plan

  variables {
    suffix = ""
    limit  = 7
  }

  assert {
    condition     = output.result == "fixture"
    error_message = "with no suffix the prefix alone must be cut to limit."
  }
}

run "fractional_limit_is_rejected" {
  command = plan

  variables {
    limit = 21.5
  }

  expect_failures = [var.limit]
}
