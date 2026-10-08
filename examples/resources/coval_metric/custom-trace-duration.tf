resource "coval_metric" "llm_span_duration" {
  metric_name        = "LLM Span Duration"
  description        = "Average duration of selected LLM spans."
  metric_type        = "METRIC_CUSTOM_TRACE"
  span_name          = "llm"
  value_source       = "duration"
  aggregation_method = "AVERAGE"
  unit               = "s"
}
