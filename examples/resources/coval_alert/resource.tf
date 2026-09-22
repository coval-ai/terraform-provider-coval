resource "coval_alert" "low_resolution" {
  name                = "Low Resolution Rate"
  description         = "Records an event when the resolution rate drops below the target."
  conversation_source = "SIMULATED"
  agent_ids           = [coval_agent.support.id]

  conditions = [
    {
      metric_id       = coval_metric.resolution.id
      aggregation     = "RUN_AVERAGE"
      operator        = "LT"
      threshold_float = 0.9
    }
  ]

  # Evaluation-only alerts do not require a notification channel.
  channels = []
}
