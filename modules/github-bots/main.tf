resource "google_service_account" "sa" {
  count        = var.service_account_email == "" ? 1 : 0
  account_id   = "bot-${var.name}"
  display_name = "Service Account for ${var.name}"
}

module "service" {
  source = "../regional-go-service"

  name       = var.name
  project_id = var.project_id
  regions    = var.regions

  labels = var.labels

  squad         = var.squad
  require_squad = var.require_squad

  service_account = var.service_account_email == "" ? google_service_account.sa[0].email : var.service_account_email

  egress = "PRIVATE_RANGES_ONLY" // Makes GitHub API calls

  containers = var.containers

  enable_profiler = var.enable_profiler

  deletion_protection = var.deletion_protection

  notification_channels = var.notification_channels
}

locals {
  combined_filter = merge(
    { "type" : var.github-event }, // Default filter setup
    var.extra_filter               // Merges any additional filters provided
  )

  combined_filter_prefix = merge(
    {},                     // Default empty map
    var.extra_filter_prefix // Merges any provided prefix filters
  )

  combined_filter_has_attributes = concat(
    [],                             // Default empty map
    var.extra_filter_has_attributes // Merges any provided filters for attributes
  )

  combined_filter_not_has_attributes = concat(
    [],                                 // Default empty map
    var.extra_filter_not_has_attributes // Merges any provided filters for not attributes
  )
}

module "cloudevent-trigger" {
  depends_on = [module.service]

  for_each = var.regions
  source   = "../cloudevent-trigger"

  project_id                = var.project_id
  name                      = "bot-trigger-${var.name}"
  broker                    = var.broker[each.key]
  raw_filter                = var.raw_filter
  filter                    = local.combined_filter
  filter_prefix             = local.combined_filter_prefix
  filter_has_attributes     = local.combined_filter_has_attributes
  filter_not_has_attributes = local.combined_filter_not_has_attributes

  private-service = {
    region = each.key
    name   = var.name
  }

  team         = var.squad
  require_team = var.require_squad

  notification_channels = var.notification_channels
}

module "dashboard" {
  source = "../dashboard/cloudevent-receiver"

  project_id   = var.project_id
  service_name = var.name

  triggers = {
    (var.name) : {
      subscription_prefix   = "bot-trigger-${var.name}"
      notification_channels = var.notification_channels
    }
  }

  notification_channels = var.notification_channels
}

output "json" {
  value = module.dashboard.json
}
