resource "coval_run_template" "regression" {
  display_name = "Support Regression"
  description  = "Runs the support regression suite against the production configuration."

  agent_ids    = [coval_agent.support.id]
  persona_ids  = [coval_persona.customer.id]
  test_set_ids = [coval_test_set.regression.id]
  metric_ids   = [coval_metric.resolution.id]

  iteration_count = 3
  concurrency     = 5

  metadata = {
    owner = "support"
  }

  tags = ["regression", "support"]
}
