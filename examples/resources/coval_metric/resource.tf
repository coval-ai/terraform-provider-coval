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

resource "coval_metric" "agentic_resolution" {
  metric_name = "Agentic Issue Resolution"
  description = "Evaluates issue resolution using transcript and trace evidence."
  metric_type = "METRIC_LLM_BINARY"
  judge_mode  = "AGENTIC"
  prompt      = "Did the agent resolve the customer's issue? Cite supporting evidence."

  enabled_tools = ["get_transcript", "search_transcript", "get_trace_spans"]
}
