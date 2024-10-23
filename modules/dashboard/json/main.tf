variable "object" {
  description = "Object to encode into JSON"
  type        = object
}

output "json" {
  value = replace(replace(replace(replace(replace(replace(replace(jsonencode({
    displayName      = "Elastic Build"
    labels           = { "elastic-builds" : "" }

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  }),
  "\"collapsibleGroup\":{\"collapsed\":false},", ""),
  ",\"xPos\":0", ""),
  ",\"yPos\":0", ""),
  ",\"thresholds\":[]", ""),
  ",\"crossSeriesReducer\":\"REDUCE_NONE\"", ""),
  ",\"perSeriesAligner\":\"ALIGN_NONE\"", ""),
  ",\"groupByFields\":[]","")
}
