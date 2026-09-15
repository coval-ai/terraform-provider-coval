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
	_ datasource.DataSource              = &testCaseDataSource{}
	_ datasource.DataSourceWithConfigure = &testCaseDataSource{}
)

type testCaseDataSource struct {
	client *client.Client
}

func newTestCaseDataSource() datasource.DataSource {
	return &testCaseDataSource{}
}

func (d *testCaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_test_case"
}

func (d *testCaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a Coval test case by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Test-case ID.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(22, 22)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Canonical API resource name.",
				Computed:            true,
			},
			"test_set_id": schema.StringAttribute{
				MarkdownDescription: "ID of the Coval test set containing this test case.",
				Computed:            true,
			},
			"input_str": schema.StringAttribute{
				MarkdownDescription: "Scenario, transcript, or other input presented by this test case.",
				Computed:            true,
			},
			"expected_behaviors": schema.ListAttribute{
				MarkdownDescription: "Ordered behaviors expected from the agent.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"expected_output_json": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing the expected structured output.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Human-readable test-case description.",
				Computed:            true,
			},
			"input_type": schema.StringAttribute{
				MarkdownDescription: "Test-case input type.",
				Computed:            true,
			},
			"script_turns": schema.DynamicAttribute{
				MarkdownDescription: "Ordered JSON array of persona turns used by SCRIPT test cases.",
				Computed:            true,
			},
			"simulation_metadata_input": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing simulation metadata.",
				Computed:            true,
			},
			"metric_input": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing metric input data.",
				Computed:            true,
			},
			"user_notes": schema.StringAttribute{
				MarkdownDescription: "User-provided notes about the test case.",
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

func (d *testCaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *testCaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config testCaseResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := d.client.GetTestCase(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval test case", err.Error())
		return
	}
	state, diagnostics := testCaseResourceState(ctx, remote, nil)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
