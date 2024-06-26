// Create a dedicated GSA for the rotating GitHub tokens.
resource "google_service_account" "octo-sts-rotator" {
  project = var.project_id

  account_id   = var.name
  display_name = "Octo STS Secret Rotator"
  description  = "Dedicated service account for rotating the Octo STS token for ${var.name}"
}

module "gh-token-secret" {
  source = "../secret"

  project_id = var.project_id
  name       = var.name

  service-account  = var.service_account
  authorized-adder = "serviceAccount:${google_service_account.octo-sts-rotator.email}"

  notification-channels = var.notification_channels
}

// The cron rotation job needs to list & delete access to cleanup old secrets.
resource "google_secret_manager_secret_iam_binding" "authorize-manage" {
  secret_id = module.gh-token-secret.secret_id
  role      = "roles/secretmanager.secretVersionManager"
  members   = ["serviceAccount:${google_service_account.octo-sts-rotator.email}"]
}

module "this" {
  source = "../cron"

  name       = var.name
  project_id = var.project_id
  region     = var.region

  invokers = var.invokers

  timeout = "60s" // 1 minute
  # Run every 30 minutes
  schedule = "*/30 * * * *"

  working_dir = path.module
  importpath  = "./cmd/rotate"

  service_account = google_service_account.octo-sts-rotator.email

  env = {
    GITHUB_ORG          = var.github_org
    GITHUB_REPO         = var.github_repo
    OCTOSTS_POLICY      = var.octosts_policy
    GITHUB_TOKEN_SECRET = module.gh-token-secret.secret_id
  }

  notification_channels = var.notification_channels
}
