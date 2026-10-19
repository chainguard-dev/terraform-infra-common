# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# The Cloud Identity API lists a nested group under its own address next to
# its members, mixes case in addresses, and repeats an identity it reaches
# through several groups. These runs pin how the module turns that into the
# member emails an audit log can name.

mock_provider "google" {
  mock_data "google_cloud_identity_group_lookup" {
    defaults = {
      name = "groups/fixture"
    }
  }
  mock_data "google_cloud_identity_group_transitive_memberships" {
    defaults = {
      memberships = []
    }
  }
}

variables {
  groups = ["oncall@fixture.dev"]
}

run "members_resolve_to_sorted_lowercase_emails" {
  command = plan

  override_data {
    target = data.google_cloud_identity_group_transitive_memberships.this["oncall@fixture.dev"]
    values = {
      memberships = [
        {
          member               = "users/2"
          preferred_member_key = [{ id = "Zoe@fixture.dev", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
        {
          member               = "users/1"
          preferred_member_key = [{ id = "alice@fixture.dev", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "OWNER", expiry_detail = [] }, { role = "MEMBER", expiry_detail = [] }]
        },
      ]
    }
  }

  assert {
    condition     = output.emails == { "oncall@fixture.dev" = tolist(["alice@fixture.dev", "zoe@fixture.dev"]) }
    error_message = "members must resolve to sorted, lowercase emails keyed by group"
  }
}

run "nested_group_contributes_its_members_not_its_address" {
  command = plan

  override_data {
    target = data.google_cloud_identity_group_transitive_memberships.this["oncall@fixture.dev"]
    values = {
      memberships = [
        {
          member               = "groups/9"
          preferred_member_key = [{ id = "oncall-secondary@fixture.dev", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
        {
          member               = "users/3"
          preferred_member_key = [{ id = "bob@fixture.dev", namespace = "" }]
          relation_type        = "INDIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
      ]
    }
  }

  assert {
    condition     = output.emails["oncall@fixture.dev"] == tolist(["bob@fixture.dev"])
    error_message = "a nested group must contribute its members, never its own address"
  }
}

run "service_account_members_are_kept" {
  command = plan

  # The API documents users/ and groups/ member names only, so a member is kept
  # by its email key rather than by its member prefix.
  override_data {
    target = data.google_cloud_identity_group_transitive_memberships.this["oncall@fixture.dev"]
    values = {
      memberships = [
        {
          member               = "users/4"
          preferred_member_key = [{ id = "rotator@fixture-project.iam.gserviceaccount.com", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
        {
          member               = "serviceAccounts/8"
          preferred_member_key = [{ id = "signer@fixture-project.iam.gserviceaccount.com", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
      ]
    }
  }

  assert {
    condition = output.emails["oncall@fixture.dev"] == tolist([
      "rotator@fixture-project.iam.gserviceaccount.com",
      "signer@fixture-project.iam.gserviceaccount.com",
    ])
    error_message = "a service account in the group acts under its own email and must be kept, whatever its member prefix"
  }
}

run "member_without_an_email_is_skipped" {
  command = plan

  override_data {
    target = data.google_cloud_identity_group_transitive_memberships.this["oncall@fixture.dev"]
    values = {
      memberships = [
        {
          member               = "cbcmBrowsers/9"
          preferred_member_key = [{ id = "0123456789abcdef", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
        {
          member               = "users/10"
          preferred_member_key = [{ id = "erin@fixture.dev", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
      ]
    }
  }

  assert {
    condition     = output.emails["oncall@fixture.dev"] == tolist(["erin@fixture.dev"])
    error_message = "a device or shared drive never acts under an email and must be skipped"
  }
}

run "an_identity_reached_twice_is_listed_once" {
  command = plan

  override_data {
    target = data.google_cloud_identity_group_transitive_memberships.this["oncall@fixture.dev"]
    values = {
      memberships = [
        {
          member               = "users/5"
          preferred_member_key = [{ id = "carol@fixture.dev", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
        {
          member               = "users/5"
          preferred_member_key = [{ id = "Carol@fixture.dev", namespace = "" }]
          relation_type        = "INDIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
      ]
    }
  }

  assert {
    condition     = output.emails["oncall@fixture.dev"] == tolist(["carol@fixture.dev"])
    error_message = "an identity reached directly and through a nested group must appear once"
  }
}

run "each_group_is_looked_up_by_its_own_address" {
  command = plan

  variables {
    groups = ["oncall@fixture.dev", "rotators@fixture.dev"]
  }

  override_data {
    target = data.google_cloud_identity_group_lookup.this["rotators@fixture.dev"]
    values = {
      name = "groups/rotators"
    }
  }

  assert {
    condition     = data.google_cloud_identity_group_lookup.this["rotators@fixture.dev"].group_key[0].id == "rotators@fixture.dev"
    error_message = "each lookup must use its group's email"
  }
  assert {
    condition     = data.google_cloud_identity_group_transitive_memberships.this["rotators@fixture.dev"].group == "groups/rotators"
    error_message = "each membership listing must use its own group's resolved name"
  }
  assert {
    condition     = toset(keys(output.emails)) == toset(["oncall@fixture.dev", "rotators@fixture.dev"])
    error_message = "every group must get an entry, even an empty one"
  }
}

run "no_groups_reads_nothing" {
  command = plan

  variables {
    groups = []
  }

  assert {
    condition     = length(data.google_cloud_identity_group_transitive_memberships.this) == 0 && length(output.emails) == 0
    error_message = "no groups must make no Cloud Identity calls"
  }
}

run "member_style_group_is_rejected" {
  command = plan

  variables {
    groups = ["group:oncall@fixture.dev"]
  }

  expect_failures = [var.groups]
}

run "uppercase_group_is_rejected" {
  command = plan

  variables {
    groups = ["OnCall@fixture.dev"]
  }

  expect_failures = [var.groups]
}

run "member_without_a_key_is_skipped" {
  command = plan

  override_data {
    target = data.google_cloud_identity_group_transitive_memberships.this["oncall@fixture.dev"]
    values = {
      memberships = [
        {
          member               = "users/6"
          preferred_member_key = []
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
        {
          member               = "users/7"
          preferred_member_key = [{ id = "dave@fixture.dev", namespace = "" }]
          relation_type        = "DIRECT"
          roles                = [{ role = "MEMBER", expiry_detail = [] }]
        },
      ]
    }
  }

  assert {
    condition     = output.emails["oncall@fixture.dev"] == tolist(["dave@fixture.dev"])
    error_message = "a member with no preferred key has no email to match and must be skipped without failing the plan"
  }
}
