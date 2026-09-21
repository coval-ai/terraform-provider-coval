package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &testSetDataSource{}
	_ datasource.DataSourceWithConfigure = &testSetDataSource{}
)

type testSetDataSource struct {
	client *client.Client
}

func newTestSetDataSource() datasource.DataSource {
	return &testSetDataSource{}
}

func (d *testSetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_test_set"
}

func (d *testSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a Coval test set by ID.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": workspaceDataSourceAttribute(),
			"id": schema.StringAttribute{
				MarkdownDescription: "Test-set ID.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(8, 8)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Canonical API resource name.",
				Computed:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "URL-friendly test-set identifier.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable test-set name.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Test-set description.",
				Computed:            true,
			},
			"test_set_type": schema.StringAttribute{
				MarkdownDescription: "Test-set type.",
				Computed:            true,
			},
			"test_set_metadata": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing additional test-set configuration.",
				Computed:            true,
			},
			"parameters": schema.DynamicAttribute{
				MarkdownDescription: "JSON object mapping parameter names to value arrays.",
				Computed:            true,
			},
			"test_case_count": schema.Int64Attribute{
				MarkdownDescription: "Number of active test cases.",
				Computed:            true,
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags associated with the test set.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 creation timestamp.",
				Computed:            true,
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 timestamp of the latest update.",
				Computed:            true,
			},
		},
	}
}

func (d *testSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *testSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config testSetResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := clientForWorkspace(d.client, config.WorkspaceID).GetTestSet(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval test set", err.Error())
		return
	}
	state, diagnostics := testSetDataSourceState(ctx, remote, &config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
