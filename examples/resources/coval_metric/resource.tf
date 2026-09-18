resource "coval_metric" "resolution" {
  metric_name = "Issue Resolution"
  description = "Determines whether the agent resolved the customer's issue."
  metric_type = "METRIC_LLM_BINARY"
  prompt      = "Did the agent resolve the customer's issue?"

  target_condition = {
    comparison_operator = "in"
    target_values       = ["YES"]
  }

  tags = ["support", "production"]
}
