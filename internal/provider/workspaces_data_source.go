package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var (
	_ datasource.DataSource              = &workspacesDataSource{}
	_ datasource.DataSourceWithConfigure = &workspacesDataSource{}
)

type workspacesDataSource struct {
	client *client.Client
}

type workspacesDataSourceModel struct {
	Workspaces []workspaceResourceModel `tfsdk:"workspaces"`
}

func newWorkspacesDataSource() datasource.DataSource {
	return &workspacesDataSource{}
}

func (d *workspacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspaces"
}

func (d *workspacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists active and archived Coval workspaces for the organization.",
		Attributes: map[string]schema.Attribute{
			"workspaces": schema.ListNestedAttribute{
				MarkdownDescription: "Organization workspaces, including the default workspace.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":              schema.StringAttribute{Computed: true},
					"display_name":    schema.StringAttribute{Computed: true},
					"status":          schema.StringAttribute{Computed: true},
					"workspace_type":  schema.StringAttribute{Computed: true},
					"created_at":      schema.StringAttribute{Computed: true},
					"last_updated_at": schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *workspacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *workspacesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	remote, err := d.client.ListWorkspaces(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval workspaces", err.Error())
		return
	}
	state := workspacesDataSourceModel{Workspaces: make([]workspaceResourceModel, len(remote))}
	for index, workspace := range remote {
		state.Workspaces[index] = workspaceState(workspace)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
