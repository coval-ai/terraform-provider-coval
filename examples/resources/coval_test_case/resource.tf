resource "coval_test_set" "regression" {
  display_name  = "Regression"
  test_set_type = "SCENARIO"
}

resource "coval_test_case" "refund_policy" {
  test_set_id = coval_test_set.regression.id
  input_str   = "Ask whether an item purchased three weeks ago can be returned."

  expected_behaviors = [
    "Explain the 30-day return policy",
    "Offer to begin the return process",
  ]

  expected_output_json = {
    eligible_for_return = true
  }

  metric_input = {
    policy_window_days = 30
  }
}
