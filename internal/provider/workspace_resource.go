package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &workspaceResource{}
	_ resource.ResourceWithConfigure   = &workspaceResource{}
	_ resource.ResourceWithImportState = &workspaceResource{}
	_ resource.ResourceWithIdentity    = &workspaceResource{}
)

type workspaceResource struct {
	client *client.Client
}

type workspaceResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Slug          types.String `tfsdk:"slug"`
	DisplayName   types.String `tfsdk:"display_name"`
	Status        types.String `tfsdk:"status"`
	WorkspaceType types.String `tfsdk:"workspace_type"`
	CreatedAt     types.String `tfsdk:"created_at"`
	LastUpdatedAt types.String `tfsdk:"last_updated_at"`
}

type workspaceIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func newWorkspaceResource() resource.Resource {
	return &workspaceResource{}
}

func (r *workspaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (r *workspaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom Coval workspace. Deletion is a permanent soft delete: the workspace disappears from reads, but its slug remains reserved.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned workspace ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "Organization-unique workspace slug. Soft-deleted workspaces continue to reserve their slugs.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`),
						"must contain lowercase letters and digits separated by single hyphens",
					),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable workspace name.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Workspace lifecycle status.",
				Computed:            true,
			},
			"workspace_type": schema.StringAttribute{
				MarkdownDescription: "Workspace type. Terraform-managed workspaces are CUSTOM.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 creation timestamp.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"last_updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 timestamp of the latest update.",
				Computed:            true,
			},
		},
	}
}

func (r *workspaceResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true},
	}}
}

func (r *workspaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return
	}
	r.client = apiClient
}

func (r *workspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workspaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.client.CreateWorkspace(ctx, client.CreateWorkspaceInput{
		Slug:        plan.Slug.ValueString(),
		DisplayName: plan.DisplayName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval workspace", err.Error())
		return
	}
	state := workspaceState(remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, workspaceIdentityModel{ID: state.ID})...)
}

func (r *workspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workspaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.client.GetWorkspace(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval workspace", err.Error())
		return
	}
	if remote.WorkspaceType != "CUSTOM" {
		resp.Diagnostics.AddError(
			"Default workspace cannot be managed",
			"Use the coval_workspace data source for a default workspace. Only custom workspaces support the complete Terraform lifecycle.",
		)
		return
	}
	refreshed := workspaceState(remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, workspaceIdentityModel{ID: refreshed.ID})...)
}

func (r *workspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan workspaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.client.UpdateWorkspace(ctx, plan.ID.ValueString(), client.UpdateWorkspaceInput{
		DisplayName: plan.DisplayName.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval workspace", err.Error())
		return
	}
	state := workspaceState(remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, workspaceIdentityModel{ID: state.ID})...)
}

func (r *workspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workspaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteWorkspace(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval workspace", err.Error())
	}
}

func (r *workspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func workspaceState(remote client.Workspace) workspaceResourceModel {
	return workspaceResourceModel{
		ID:            types.StringValue(remote.ID),
		Slug:          types.StringValue(remote.Slug),
		DisplayName:   types.StringValue(remote.DisplayName),
		Status:        types.StringValue(remote.Status),
		WorkspaceType: types.StringValue(remote.WorkspaceType),
		CreatedAt:     types.StringValue(remote.CreatedAt),
		LastUpdatedAt: types.StringValue(remote.LastUpdatedAt),
	}
}
