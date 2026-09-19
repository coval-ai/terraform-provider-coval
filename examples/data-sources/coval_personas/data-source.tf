data "coval_personas" "support" {
  filter      = "name=\"Friendly Support Customer\""
  order_by    = "-create_time"
  tag_filters = ["support", "voice"]
}
