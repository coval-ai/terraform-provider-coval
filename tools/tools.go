//go:build generate

package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

// Format Terraform examples before rendering them in documentation.
//go:generate terraform fmt -recursive ../examples/

// Render provider documentation from the schema and examples.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-dir .. -provider-name coval
