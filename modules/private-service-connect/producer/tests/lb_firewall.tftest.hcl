# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Applies against the mock so the forwarding rule's self-link is known when the
# rules are checked.
mock_provider "google" {}

override_data {
  target = data.google_compute_subnetwork.psc_nat
  values = {
    ip_cidr_range = "192.0.2.0/24"
  }
}

variables {
  project                  = "fixture-project"
  region                   = "us-central1"
  name                     = "fixture"
  network                  = "projects/fixture-project/global/networks/fixture"
  subnetwork               = "projects/fixture-project/regions/us-central1/subnetworks/frontend"
  proxy_only_subnet        = "projects/fixture-project/regions/us-central1/subnetworks/proxy"
  psc_nat_subnets          = ["projects/fixture-project/regions/us-central1/subnetworks/psc-nat"]
  cloud_run_service_name   = "fixture"
  consumer_accept_projects = ["consumer-project"]
}

run "no_rules_without_a_policy" {
  command = plan

  assert {
    condition     = length(google_compute_region_network_firewall_policy_rule.lb_allow) == 0 && length(google_compute_region_network_firewall_policy_rule.lb_deny) == 0
    error_message = "Without lb_firewall the module must add no firewall rules."
  }
}

# Envoy-based load balancers admit every client in the network when no rule
# targets them, so the deny below the allow is what closes the VIP.
run "rules_admit_only_psc_nat_to_this_forwarding_rule" {
  command = apply

  variables {
    lb_firewall = {
      policy   = "fixture-policy"
      priority = 1000
    }
  }

  assert {
    condition = (
      google_compute_region_network_firewall_policy_rule.lb_allow[0].action == "allow" &&
      google_compute_region_network_firewall_policy_rule.lb_allow[0].direction == "INGRESS" &&
      google_compute_region_network_firewall_policy_rule.lb_allow[0].target_type == "INTERNAL_MANAGED_LB" &&
      toset(google_compute_region_network_firewall_policy_rule.lb_allow[0].target_forwarding_rules) == toset([google_compute_forwarding_rule.this.self_link]) &&
      toset(google_compute_region_network_firewall_policy_rule.lb_allow[0].match[0].src_ip_ranges) == toset(["192.0.2.0/24"]) &&
      tolist(google_compute_region_network_firewall_policy_rule.lb_allow[0].match[0].layer4_configs[0].ports) == tolist(["80"]) &&
      google_compute_region_network_firewall_policy_rule.lb_allow[0].priority == 1000
    )
    error_message = "The allow rule must admit only the PSC NAT ranges to this forwarding rule on its frontend port."
  }

  assert {
    condition = (
      google_compute_region_network_firewall_policy_rule.lb_deny[0].action == "deny" &&
      google_compute_region_network_firewall_policy_rule.lb_deny[0].target_type == "INTERNAL_MANAGED_LB" &&
      toset(google_compute_region_network_firewall_policy_rule.lb_deny[0].target_forwarding_rules) == toset([google_compute_forwarding_rule.this.self_link]) &&
      toset(google_compute_region_network_firewall_policy_rule.lb_deny[0].match[0].src_ip_ranges) == toset(["0.0.0.0/0"]) &&
      google_compute_region_network_firewall_policy_rule.lb_deny[0].priority == 1001
    )
    error_message = "The deny rule must close this forwarding rule to everything else, just below the allow."
  }

  assert {
    condition     = google_compute_region_network_firewall_policy_rule.lb_allow[0].project == "fixture-project" && google_compute_region_network_firewall_policy_rule.lb_deny[0].project == "fixture-project"
    error_message = "Rules must default to the producer project's policy."
  }
}

run "https_frontend_and_extra_sources_in_a_host_project_policy" {
  command = apply

  variables {
    tls_certificates = ["projects/fixture-project/locations/us-central1/certificates/fixture"]
    lb_firewall = {
      policy                = "host-policy"
      project               = "host-project"
      priority              = 2000
      allowed_source_ranges = ["198.51.100.0/24"]
    }
  }

  assert {
    condition = (
      tolist(google_compute_region_network_firewall_policy_rule.lb_allow[0].match[0].layer4_configs[0].ports) == tolist(["443"]) &&
      toset(google_compute_region_network_firewall_policy_rule.lb_allow[0].match[0].src_ip_ranges) == toset(["192.0.2.0/24", "198.51.100.0/24"]) &&
      google_compute_region_network_firewall_policy_rule.lb_allow[0].project == "host-project" &&
      google_compute_region_network_firewall_policy_rule.lb_deny[0].project == "host-project"
    )
    error_message = "The allow rule must follow the frontend port and extra sources, in the named policy project."
  }
}

run "priority_must_leave_room_for_the_deny" {
  command = plan

  variables {
    lb_firewall = {
      policy   = "fixture-policy"
      priority = 2147483547
    }
  }

  expect_failures = [var.lb_firewall]
}
