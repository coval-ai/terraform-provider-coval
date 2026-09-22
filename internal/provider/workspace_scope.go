package provider

import (
	"context"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func workspaceResourceAttribute() resourceschema.StringAttribute {
	return resourceschema.StringAttribute{
		MarkdownDescription: "Workspace ID that owns this resource. Omit it to use the organization's active default workspace.",
		Optional:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		Validators:          []validator.String{stringvalidator.LengthBetween(1, 26)},
	}
}

func workspaceDataSourceAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "Workspace ID that scopes this lookup. Omit it to use the organization's active default workspace.",
		Optional:            true,
		Validators:          []validator.String{stringvalidator.LengthBetween(1, 26)},
	}
}

func clientForWorkspace(apiClient *client.Client, workspaceID types.String) *client.Client {
	if workspaceID.IsNull() || workspaceID.IsUnknown() {
		return apiClient
	}
	return apiClient.ForWorkspace(workspaceID.ValueString())
}

func importStateWithWorkspaceIdentity(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
	if req.ID == "" && !resp.Diagnostics.HasError() {
		resource.ImportStatePassthroughWithIdentity(
			ctx,
			path.Root("workspace_id"),
			path.Root("workspace_id"),
			req,
			resp,
		)
	}
}
