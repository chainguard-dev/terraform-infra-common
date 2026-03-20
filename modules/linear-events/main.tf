resource "random_string" "service-suffix" {
  length  = 4
  upper   = false
  special = false
}

// A dedicated service account for the trampoline service.
resource "google_service_account" "service" {
  project = var.project_id

  account_id   = "${var.name}-${random_string.service-suffix.result}"
  display_name = "Service account for Linear events trampoline service"
}

module "webhook-secret" {
  source = "../secret"

  project_id = var.project_id
  name       = "${var.name}-webhook-secret"

  service-accounts = compact([
    google_service_account.service.email,
    # Allow the provisioning account to access the secret
    # values because we're creating a placeholder
    var.provisioner,
  ])
  authorized-adder = var.secret_version_adder

  create_placeholder_version = var.create_placeholder_version

  notification-channels = var.notification_channels

  team = var.team
}

module "this" {
  source     = "../regional-go-service"
  project_id = var.project_id
  name       = var.name
  regions    = var.regions

  ingress = var.service-ingress

  deletion_protection = var.deletion_protection

  team            = var.team
  service_account = google_service_account.service.email
  containers = {
    "trampoline" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/trampoline"
      }
      ports = [{ container_port = 8080 }]
      env = concat(
        [
          {
            name = "WEBHOOK_SECRET"
            value_source = {
              secret_key_ref = {
                secret  = module.webhook-secret.secret_id
                version = "latest"
              }
            }
          },
        ],
        [for name, secret in var.additional_webhook_secrets : {
          name = "WEBHOOK_SECRET_${upper(name)}"
          value_source = {
            secret_key_ref = {
              secret  = secret.secret
              version = secret.version
            }
          }
        }],
      )

      regional-env = [{
        name  = "EVENT_INGRESS_URI"
        value = { for k, v in module.trampoline-emits-events : k => v.uri }
      }]
    }
  }

  enable_profiler = var.enable_profiler

  notification_channels = var.notification_channels
}

// Authorize the trampoline service account to publish events on the broker.
module "trampoline-emits-events" {
  for_each = var.regions
  source   = "../authorize-private-service"

  project_id = var.project_id
  region     = each.key
  name       = var.ingress.name

  service-account = google_service_account.service.email
}

data "google_cloud_run_v2_service" "this" {
  for_each   = var.service-ingress == "INGRESS_TRAFFIC_ALL" ? var.regions : {}
  project    = var.project_id
  location   = each.key
  name       = var.name
  depends_on = [module.this]
}

output "public-urls" {
  description = "Map of region to public URL for the service, if service-ingress is INGRESS_TRAFFIC_ALL."
  value = var.service-ingress == "INGRESS_TRAFFIC_ALL" ? {
    for r, _ in var.regions : r => data.google_cloud_run_v2_service.this[r].uri
  } : {}
}

output "recorder-schemas" {
  value = {
    "dev.chainguard.linear.issue" : {
      schema = file("${path.module}/schemas/issue.schema.json")
    }
    "dev.chainguard.linear.comment" : {
      schema = file("${path.module}/schemas/comment.schema.json")
    }
    "dev.chainguard.linear.issuelabel" : {
      schema = file("${path.module}/schemas/issuelabel.schema.json")
    }
    "dev.chainguard.linear.attachment" : {
      schema = file("${path.module}/schemas/attachment.schema.json")
    }
    "dev.chainguard.linear.reaction" : {
      schema = file("${path.module}/schemas/reaction.schema.json")
    }
    "dev.chainguard.linear.project" : {
      schema = file("${path.module}/schemas/project.schema.json")
    }
    "dev.chainguard.linear.projectupdate" : {
      schema = file("${path.module}/schemas/projectupdate.schema.json")
    }
    "dev.chainguard.linear.document" : {
      schema = file("${path.module}/schemas/document.schema.json")
    }
    "dev.chainguard.linear.initiative" : {
      schema = file("${path.module}/schemas/initiative.schema.json")
    }
    "dev.chainguard.linear.cycle" : {
      schema = file("${path.module}/schemas/cycle.schema.json")
    }
    "dev.chainguard.linear.customer" : {
      schema = file("${path.module}/schemas/customer.schema.json")
    }
    "dev.chainguard.linear.customerneed" : {
      schema = file("${path.module}/schemas/customerneed.schema.json")
    }
    "dev.chainguard.linear.issuesla" : {
      schema = file("${path.module}/schemas/issuesla.schema.json")
    }
  }
}
