resource "coval_scheduled_run" "nightly" {
  display_name        = "Nightly Regression"
  run_template_id     = coval_run_template.regression.id
  schedule_expression = "cron(0 2 ? * * *)"
  schedule_timezone   = "UTC"

  # Keep a new schedule disabled until its configuration has been reviewed.
  enabled = false
}
