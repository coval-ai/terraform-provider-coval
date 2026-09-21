resource "coval_workspace" "example" {
  slug         = "example"
  display_name = "Example Workspace"
}

resource "coval_test_set" "smoke" {
  workspace_id = coval_workspace.example.id
  display_name = "Smoke Tests"
  description  = "Critical-path scenarios managed by Terraform"
}
