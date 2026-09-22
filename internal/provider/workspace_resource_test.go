package provider

import (
	"context"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWorkspaceResourceSchemaMatchesPublicContract(t *testing.T) {
	t.Parallel()

	var response resource.SchemaResponse
	newWorkspaceResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", response.Diagnostics)
	}

	if _, ok := response.Schema.Attributes["slug"]; ok {
		t.Error("workspace resource must not expose the deprecated public API slug")
	}

	displayName, ok := response.Schema.Attributes["display_name"].(schema.StringAttribute)
	if !ok || !displayName.Required {
		t.Fatalf("display_name must be a required string: %#v", response.Schema.Attributes["display_name"])
	}
	for _, name := range []string{"status", "workspace_type", "created_at", "last_updated_at"} {
		attribute, ok := response.Schema.Attributes[name].(schema.StringAttribute)
		if !ok || !attribute.Computed {
			t.Errorf("%s must be a computed string: %#v", name, response.Schema.Attributes[name])
		}
	}

	var identityResponse resource.IdentitySchemaResponse
	identityResource, ok := newWorkspaceResource().(resource.ResourceWithIdentity)
	if !ok {
		t.Fatal("workspace resource does not implement ResourceWithIdentity")
	}
	identityResource.IdentitySchema(
		context.Background(),
		resource.IdentitySchemaRequest{},
		&identityResponse,
	)
	id, ok := identityResponse.IdentitySchema.Attributes["id"].(identityschema.StringAttribute)
	if !ok || !id.RequiredForImport {
		t.Fatalf("workspace identity id must be required for import: %#v", identityResponse.IdentitySchema.Attributes["id"])
	}
}

func TestWorkspaceStateMapsPublicAPIResponse(t *testing.T) {
	t.Parallel()

	remote := client.Workspace{
		ID:            "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		DisplayName:   "Example Workspace",
		Status:        "ACTIVE",
		WorkspaceType: "CUSTOM",
		CreatedAt:     "2026-09-21T00:00:00Z",
		LastUpdatedAt: "2026-09-21T01:00:00Z",
	}
	state := workspaceState(remote)

	if got, want := state.ID, types.StringValue(remote.ID); !got.Equal(want) {
		t.Errorf("id = %s, want %s", got, want)
	}
	if got, want := state.DisplayName, types.StringValue(remote.DisplayName); !got.Equal(want) {
		t.Errorf("display_name = %s, want %s", got, want)
	}
	if got, want := state.Status, types.StringValue(remote.Status); !got.Equal(want) {
		t.Errorf("status = %s, want %s", got, want)
	}
	if got, want := state.WorkspaceType, types.StringValue(remote.WorkspaceType); !got.Equal(want) {
		t.Errorf("workspace_type = %s, want %s", got, want)
	}
	if got, want := state.CreatedAt, types.StringValue(remote.CreatedAt); !got.Equal(want) {
		t.Errorf("created_at = %s, want %s", got, want)
	}
	if got, want := state.LastUpdatedAt, types.StringValue(remote.LastUpdatedAt); !got.Equal(want) {
		t.Errorf("last_updated_at = %s, want %s", got, want)
	}
}
