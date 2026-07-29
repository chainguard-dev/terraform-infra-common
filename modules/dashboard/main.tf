variable "object" {
  description = "Object to encode into JSON"
}

locals {
  json = replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(
    jsonencode(var.object),
    "\"collapsed\":false", ""),
    ",\"xPos\":0", ""),
    ",\"yPos\":0", ""),
    ",\"thresholds\":[]", ""),
    ",\"crossSeriesReducer\":\"REDUCE_NONE\"", ""),
    ",\"perSeriesAligner\":\"ALIGN_NONE\"", ""),
    "\"dashboardFilters\":[],", ""),
    ",\"groupByFields\":[]", ""),
    ",\"secondaryAggregation\":null", ""),
    "\"secondaryAggregation\":null,", ""),
    "\"secondaryAggregation\":null", ""),
    ",\"groupByFields\":null", ""),
  "\"groupByFields\":null,", "")

  // GCP rejects any dashboard with more than 50 widgets, and this module is the
  // single choke point every dashboard in the repo renders through. Gather the
  // widget objects from whichever layout the dashboard uses (mosaic is by far
  // the common one; grid/row/column are handled too so a non-mosaic dashboard
  // can't slip past the cap), then count only leaf chart widgets: collapsibleGroup
  // section headers do not count toward GCP's limit, so exclude them.
  dashboard_widgets = concat(
    [for tile in try(var.object.mosaicLayout.tiles, []) : try(tile.widget, null)],
    try(var.object.gridLayout.widgets, []),
    flatten([for row in try(var.object.rowLayout.rows, []) : try(row.widgets, [])]),
    flatten([for col in try(var.object.columnLayout.columns, []) : try(col.widgets, [])]),
  )
  widget_count = length([
    for widget in local.dashboard_widgets : true
    if widget != null && try(widget.collapsibleGroup, null) == null
  ])
}

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = local.json

  lifecycle {
    precondition {
      condition     = local.widget_count <= 50
      error_message = "Dashboard '${try(var.object.displayName, "(unnamed)")}' renders ${local.widget_count} widgets, exceeding Cloud Monitoring's hard limit of 50 per dashboard. Move a section onto its own dedicated dashboard (see the microvm/agents split in modules/dashboard/reconciler/dashboard.tf)."
    }
    precondition {
      condition     = length(try(var.object.labels, {})) <= 64
      error_message = "Dashboard '${try(var.object.displayName, "(unnamed)")}' declares ${length(try(var.object.labels, {}))} labels, exceeding Cloud Monitoring's hard limit of 64 per dashboard. Stop generating a label per item (e.g. one per event type) and use a small bounded set instead."
    }
  }
}

output "json" {
  value = local.json
}
