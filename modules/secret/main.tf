terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

// Create the GCP secret to hold the configuration data.
resource "google_secret_manager_secret" "this" {
  secret_id = var.name
  labels    = local.merged_labels
  replication {
    dynamic "auto" {
      for_each = var.replication_locations == null ? ["auto"] : []
      content {}
    }
    dynamic "user_managed" {
      for_each = var.replication_locations != null ? ["user_managed"] : []
      content {
        dynamic "replicas" {
          for_each = var.replication_locations
          content {
            location = replicas.value
          }
        }
      }
    }
  }
  lifecycle {
    prevent_destroy = true
  }
}

// Create a placeholder GCP secret version to avoid bad reference on first deploy.
resource "google_secret_manager_secret_version" "placeholder" {
  count = var.create_placeholder_version ? 1 : 0

  secret      = google_secret_manager_secret.this.id
  secret_data = "placeholder"

  # We don't prevent this version from being destroyed,
  # since it is meant as a bootstrapping step when creating new
  # secrets. Once populated with a legitimate value, this
  # placeholder is no longer needed.
}

locals {
  # only append `serviceAccount:` if there is no `:` in the entry
  accessors = [for sa in concat([var.service-account], var.service-accounts) : "serviceAccount:${sa}" if sa != "" && !strcontains(sa, ":")]
  # extract the email component if there is a `:` in the entry, and choose the last entry by reversing the list
  accessor_emails = [for sa in concat([var.service-account], var.service-accounts) : reverse(split(":", sa))[0] if sa != ""]

  # Combine authorized-adder (singular) with authorized-adders (plural)
  all_authorized_adders = concat([var.authorized-adder], var.authorized-adders)

  # Audit logs name the identity that acted, never a group it acted through,
  # and a domain never acts, so only user and service-account emails can match.
  # A group's members match through var.group_members. The filter compares
  # strings without regard to case, so lowercasing only removes duplicates.
  user_email_pattern = "^(?:(?:user|serviceAccount):)?([^:]+@[^:]+)$"
  version_management_emails = sort(distinct([for e in concat(
    [for m in concat(local.all_authorized_adders, var.version_managers) : regex(local.user_email_pattern, m)[0] if can(regex(local.user_email_pattern, m))],
    flatten(values(var.group_members)),
  ) : lower(e)]))
  payload_reader_emails = sort(distinct([for e in local.accessor_emails : lower(e)]))

  version_management_methods = [
    for method in ["AddSecretVersion", "EnableSecretVersion", "DisableSecretVersion", "DestroySecretVersion"] :
    "google.cloud.secretmanager.v1.SecretManagerService.${method}"
  ]

  # Cloud Monitoring rejects a longer log-match filter, and only at apply.
  max_alert_filter_length = 20000
  alert_filter            = <<EOT
      -- This looks at logs from both data_access and activity, so we don't filter on either here.
      protoPayload.serviceName="secretmanager.googleapis.com"
      (
        protoPayload.request.name: ("projects/${var.project_id}/secrets/${var.name}/" OR "projects/${data.google_project.project.number}/secrets/${var.name}/") OR
        protoPayload.request.parent=("projects/${var.project_id}/secrets/${var.name}" OR "projects/${data.google_project.project.number}/secrets/${var.name}")
      )
%{if length(local.payload_reader_emails) > 0~}
      -- Ignore the identities that are intended to read this secret.
      -(
        protoPayload.authenticationInfo.principalEmail=(${join(" OR ", [for e in local.payload_reader_emails : "\"${e}\""])})
        protoPayload.methodName="google.cloud.secretmanager.v1.SecretManagerService.AccessSecretVersion"
      )
%{endif~}
%{if length(local.version_management_emails) > 0~}
      -- Ignore the identities that manage this secret's versions, including
      -- the members of any group that does.
      -(
        protoPayload.authenticationInfo.principalEmail=(${join(" OR ", [for e in local.version_management_emails : "\"${e}\""])})
        protoPayload.methodName=(${join(" OR ", [for m in local.version_management_methods : "\"${m}\""])})
      )
%{endif~}
      -- Ignore benign secret actions. ListSecretVersions and GetSecretVersion
      -- only return version metadata (not the secret payload), so we don't care
      -- who calls them.
      -(
        protoPayload.methodName=~"google.cloud.secretmanager.v1.SecretManagerService.(ListSecretVersions|GetSecretVersion)"
      )
      EOT

  default_labels = {
    basename(abspath(path.module)) = var.name
    terraform-module               = basename(abspath(path.module))
  }

  squad_label = {
    squad = var.team
  }
  merged_labels = merge(local.default_labels, local.squad_label, var.labels)
}

// Only the service account as which the service runs should have access to the secret.
resource "google_secret_manager_secret_iam_binding" "authorize-service-access" {
  secret_id = google_secret_manager_secret.this.id
  role      = "roles/secretmanager.secretAccessor"
  members   = local.accessors
}

// Authorize the specified identities to add new secret values.
resource "google_secret_manager_secret_iam_binding" "authorize-version-adder" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.this.secret_id
  role      = "roles/secretmanager.secretVersionAdder"
  members   = local.all_authorized_adders
}

// Authorize the version managers to add, list, get, enable, disable, and
// destroy versions and to view the secret, but not to read version payloads.
resource "google_secret_manager_secret_iam_binding" "authorize-version-manager" {
  for_each = toset(length(var.version_managers) > 0 ? ["roles/secretmanager.secretVersionManager", "roles/secretmanager.viewer"] : [])

  project   = var.project_id
  secret_id = google_secret_manager_secret.this.secret_id
  role      = each.key
  members   = var.version_managers
}

// Get a project number for this project ID.
data "google_project" "project" { project_id = var.project_id }

// Create an alert policy to notify if the secret is accessed by an unauthorized entity.
resource "google_monitoring_alert_policy" "anomalous-secret-access" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Abnormal Secret Access: ${var.name}"
  combiner     = "OR"
  user_labels  = local.merged_labels
  severity     = "WARNING"

  conditions {
    display_name = "Abnormal Secret Access: ${var.name}"

    condition_matched_log {
      filter = local.alert_filter

      label_extractors = {
        "email"       = "EXTRACT(protoPayload.authenticationInfo.principalEmail)"
        "method_name" = "EXTRACT(protoPayload.methodName)"
        "user_agent"  = "REGEXP_EXTRACT(protoPayload.requestMetadata.callerSuppliedUserAgent, \"(\\\\S+)\")"
      }
    }
  }

  dynamic "documentation" {
    for_each = var.alert_documentation == "" ? [] : [var.alert_documentation]
    content {
      content   = documentation.value
      mime_type = "text/markdown"
    }
  }

  notification_channels = var.notification-channels

  enabled = "true"
  project = var.project_id

  lifecycle {
    precondition {
      condition     = length(local.alert_filter) <= local.max_alert_filter_length
      error_message = "The abnormal-access alert filter is ${length(local.alert_filter)} characters; Cloud Monitoring rejects more than ${local.max_alert_filter_length}. Trim group_members or split the managing group."
    }
  }
}
