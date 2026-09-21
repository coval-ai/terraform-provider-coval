package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ datasource.DataSource              = &workspaceDataSource{}
	_ datasource.DataSourceWithConfigure = &workspaceDataSource{}
)

type workspaceDataSource struct {
	client *client.Client
}

func newWorkspaceDataSource() datasource.DataSource {
	return &workspaceDataSource{}
}

func (d *workspaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (d *workspaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves an active or archived Coval workspace by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workspace ID.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 26)},
			},
			"slug":            schema.StringAttribute{MarkdownDescription: "Workspace slug.", Computed: true},
			"display_name":    schema.StringAttribute{MarkdownDescription: "Human-readable workspace name.", Computed: true},
			"status":          schema.StringAttribute{MarkdownDescription: "Workspace lifecycle status.", Computed: true},
			"workspace_type":  schema.StringAttribute{MarkdownDescription: "Workspace type.", Computed: true},
			"created_at":      schema.StringAttribute{MarkdownDescription: "RFC 3339 creation timestamp.", Computed: true},
			"last_updated_at": schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest update.", Computed: true},
		},
	}
}

func (d *workspaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return
	}
	d.client = apiClient
}

func (d *workspaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config workspaceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := d.client.GetWorkspace(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval workspace", err.Error())
		return
	}
	state := workspaceState(remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
