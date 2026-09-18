data "coval_agents" "production" {
  filter      = "model_type=MODEL_TYPE_CHAT"
  tag_filters = ["production"]
  order_by    = "display_name"
}
