resource "coval_persona" "support_customer" {
  name           = "Friendly Support Customer"
  persona_prompt = "You are a friendly customer calling for technical support."
  voice_name     = "aria"
  language_code  = "en-US"

  conversation_initiation = "speak_first"
  interruption_rate       = "LOW"
  wait_seconds            = 0.5

  initialization_parameters = {
    customer_tier = "premium"
  }

  tags = ["support", "voice"]
}
