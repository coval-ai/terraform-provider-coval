data "coval_test_cases" "regression" {
  filter   = "test_set_id=\"abc12345\""
  order_by = "-create_time"
}
