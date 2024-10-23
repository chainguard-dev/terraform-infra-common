variable "object" {
  description = "Object to encode into JSON"
  type        = object({})
}

output "json" {
  value = replace(replace(replace(replace(replace(replace(replace(
    jsonencode(var.object),
    "\"collapsibleGroup\":{\"collapsed\":false},", ""),
    ",\"xPos\":0", ""),
    ",\"yPos\":0", ""),
    ",\"thresholds\":[]", ""),
    ",\"crossSeriesReducer\":\"REDUCE_NONE\"", ""),
    ",\"perSeriesAligner\":\"ALIGN_NONE\"", ""),
  ",\"groupByFields\":[]", "")
}
