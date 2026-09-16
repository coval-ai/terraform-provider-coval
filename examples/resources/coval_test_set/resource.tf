resource "coval_test_set" "support" {
  display_name  = "Customer Support Scenarios"
  description   = "Scenarios managed by Terraform"
  test_set_type = "SCENARIO"

  test_set_metadata = {
    category = "support"
    priority = "high"
  }

  parameters = {
    customer_name = ["Alice", "Bob"]
    issue_type    = ["billing", "technical"]
  }

  tags = ["regression", "voice"]
}
