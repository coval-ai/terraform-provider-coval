data "coval_metrics" "production" {
  filter          = "metric_type=METRIC_LLM_BINARY"
  tag_filters     = ["production"]
  order_by        = "metric_name asc"
  include_builtin = false
}
