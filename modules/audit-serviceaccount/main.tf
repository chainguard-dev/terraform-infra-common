resource "google_monitoring_alert_policy" "this" {
  # In the absence of data, incident will auto-close after an hour
  alert_strategy {
    auto_close = "3600s"

    notification_rate_limit {
      period = "3600s" // re-alert hourly if condition still valid.
    }
  }

  display_name = "Abnormal GSA Access: ${var.service-account}"
  combiner     = "OR"

  conditions {
    display_name = "Abnormal GSA Access"

    condition_matched_log {
      filter = <<EOT
      protoPayload.request.name="projects/-/serviceAccounts/${var.service-account}"
      protoPayload.serviceName="iamcredentials.googleapis.com"
      protoPayload.methodName="GenerateAccessToken"

      -- Allow these principals to generate tokens.
      ${join("\n", [for principal in var.allowed_principals : "-protoPayload.authenticationInfo.principalSubject=\"${principal}\""])}
      ${var.allowed_principal_regex != "" ? "-protoPayload.authenticationInfo.principalSubject=~\"${var.allowed_principal_regex}\"" : ""}
      EOT
    }
  }

  conditions {
    display_name = "Private Key Created"

    condition_matched_log {
      filter = <<EOT
      protoPayload.serviceName="iam.googleapis.com"
      protoPayload.request.name="projects/-/serviceAccounts/${var.service-account}"
      protoPayload.methodName="google.iam.admin.v1.CreateServiceAccountKey"
      EOT
    }
  }

  notification_channels = var.notification_channels

  enabled = "true"
  project = var.project_id
}
