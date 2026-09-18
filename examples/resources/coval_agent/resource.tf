resource "coval_agent" "support" {
  display_name = "Customer Support"
  model_type   = "MODEL_TYPE_CHAT"
  prompt       = "Help customers with product questions."

  metadata = {
    chat_endpoint = "https://api.example.com/chat"
  }

  tags = ["support", "production"]
}
