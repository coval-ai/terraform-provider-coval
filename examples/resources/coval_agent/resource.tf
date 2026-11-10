resource "coval_agent" "support" {
  display_name = "Customer Support"
  model_type   = "MODEL_TYPE_CHAT"
  prompt       = "Help customers with product questions."

  metadata = {
    chat_endpoint = "https://api.example.com/chat"
  }

  tags = ["support", "production"]
}

resource "coval_agent" "sdk" {
  display_name = "SDK Voice Agent"
  model_type   = "MODEL_TYPE_SDK_WEBSOCKET"
  metadata     = {}
}

resource "coval_agent" "sdk_chat" {
  display_name = "SDK Text Agent"
  model_type   = "MODEL_TYPE_SDK_CHAT"
  metadata     = {}
}
