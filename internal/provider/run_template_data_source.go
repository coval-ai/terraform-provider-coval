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
	_ datasource.DataSource              = &runTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &runTemplateDataSource{}
)

type runTemplateDataSource struct {
	client *client.Client
}

func newRunTemplateDataSource() datasource.DataSource {
	return &runTemplateDataSource{}
}

func (d *runTemplateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run_template"
}

func (d *runTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a Coval run template by ID.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": workspaceDataSourceAttribute(),
			"id": schema.StringAttribute{
				MarkdownDescription: "Run-template ID.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(22, 22)},
			},
			"name":               schema.StringAttribute{MarkdownDescription: "Canonical API resource name.", Computed: true},
			"display_name":       schema.StringAttribute{MarkdownDescription: "Human-readable run-template name.", Computed: true},
			"description":        schema.StringAttribute{MarkdownDescription: "Run-template description.", Computed: true},
			"agent_ids":          schema.SetAttribute{MarkdownDescription: "Agent IDs included in the template.", ElementType: types.StringType, Computed: true},
			"persona_ids":        schema.SetAttribute{MarkdownDescription: "Persona IDs included in the template.", ElementType: types.StringType, Computed: true},
			"test_set_ids":       schema.SetAttribute{MarkdownDescription: "Test-set IDs included in the template.", ElementType: types.StringType, Computed: true},
			"metric_ids":         schema.SetAttribute{MarkdownDescription: "Metric IDs evaluated by the template.", ElementType: types.StringType, Computed: true},
			"mutation_ids":       schema.SetAttribute{MarkdownDescription: "Mutation IDs used for A/B testing.", ElementType: types.StringType, Computed: true},
			"iteration_count":    schema.Int64Attribute{MarkdownDescription: "Number of times each test case runs.", Computed: true},
			"concurrency":        schema.Int64Attribute{MarkdownDescription: "Maximum number of simulations run concurrently.", Computed: true},
			"sub_sample_size":    schema.Int64Attribute{MarkdownDescription: "Number of test cases sampled randomly.", Computed: true},
			"sub_sample_seed":    schema.Int64Attribute{MarkdownDescription: "Random seed for reproducible sub-sampling.", Computed: true},
			"metadata":           schema.DynamicAttribute{MarkdownDescription: "Arbitrary JSON object containing run-template metadata.", Computed: true},
			"tags":               schema.SetAttribute{MarkdownDescription: "Tags associated with the run template.", ElementType: types.StringType, Computed: true},
			"create_time":        schema.StringAttribute{MarkdownDescription: "RFC 3339 creation timestamp.", Computed: true},
			"update_time":        schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest update.", Computed: true},
			"created_by_user_id": schema.StringAttribute{MarkdownDescription: "ID of the user who created the run template when available.", Computed: true},
		},
	}
}

func (d *runTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *runTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runTemplateResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := clientForWorkspace(d.client, config.WorkspaceID).GetRunTemplate(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval run template", err.Error())
		return
	}
	state, diagnostics := runTemplateDataSourceState(ctx, remote, &config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
