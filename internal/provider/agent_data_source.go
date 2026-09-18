package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &agentDataSource{}
	_ datasource.DataSourceWithConfigure = &agentDataSource{}
)

type agentDataSource struct {
	client *client.Client
}

func newAgentDataSource() datasource.DataSource {
	return &agentDataSource{}
}

func (d *agentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (d *agentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a Coval agent by ID.",
		Attributes: map[string]schema.Attribute{
			"id":                 schema.StringAttribute{MarkdownDescription: "Agent ID.", Required: true},
			"customer_agent_id":  schema.StringAttribute{MarkdownDescription: "Customer-defined external identifier.", Computed: true},
			"display_name":       schema.StringAttribute{MarkdownDescription: "Human-readable agent name.", Computed: true},
			"model_type":         schema.StringAttribute{MarkdownDescription: "Agent simulator type.", Computed: true},
			"phone_number":       schema.StringAttribute{MarkdownDescription: "Phone number or SIP address.", Computed: true},
			"endpoint":           schema.StringAttribute{MarkdownDescription: "Webhook endpoint URL.", Computed: true},
			"prompt":             schema.StringAttribute{MarkdownDescription: "Agent instructions or system prompt.", Computed: true},
			"language":           schema.StringAttribute{MarkdownDescription: "Primary language for the agent.", Computed: true},
			"attributes":         schema.DynamicAttribute{MarkdownDescription: "Free-form JSON agent attributes.", Computed: true},
			"metadata":           schema.DynamicAttribute{MarkdownDescription: "Simulator-specific JSON configuration.", Computed: true, Sensitive: true},
			"workflows":          schema.DynamicAttribute{MarkdownDescription: "Workflow JSON configuration.", Computed: true},
			"metric_ids":         schema.SetAttribute{MarkdownDescription: "Associated metric IDs.", ElementType: types.StringType, Computed: true},
			"test_set_ids":       schema.SetAttribute{MarkdownDescription: "Associated test-set IDs.", ElementType: types.StringType, Computed: true},
			"knowledge_base_ids": schema.SetAttribute{MarkdownDescription: "Associated knowledge-base entry IDs.", ElementType: types.StringType, Computed: true},
			"tags":               schema.SetAttribute{MarkdownDescription: "Tags associated with the agent.", ElementType: types.StringType, Computed: true},
			"create_time":        schema.StringAttribute{MarkdownDescription: "RFC 3339 creation timestamp.", Computed: true},
			"update_time":        schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest update.", Computed: true},
		},
	}
}

func (d *agentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *agentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config agentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := d.client.GetAgent(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval agent", err.Error())
		return
	}
	state, diagnostics := agentDataSourceState(ctx, remote)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
