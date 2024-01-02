resource "google_cloud_run_v2_service_iam_member" "authorize-calls" {
  project  = var.project_id
  location = var.region
  name     = var.name

  role   = "roles/run.invoker"
  member = "serviceAccount:${var.service-account}"
}

data "google_cloud_run_v2_service" "this" {
  depends_on = [google_cloud_run_v2_service_iam_member.authorize-calls]

  project  = var.project_id
  location = var.region
  name     = var.name
}
