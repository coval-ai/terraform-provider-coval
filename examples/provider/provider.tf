terraform {
  required_version = ">= 1.12.0"

  required_providers {
    coval = {
      source  = "coval-ai/coval"
      version = "~> 0.1"
    }
  }
}

provider "coval" {
  api_key = var.coval_api_key
}

variable "coval_api_key" {
  type      = string
  sensitive = true
}
