package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &agentsDataSource{}
	_ datasource.DataSourceWithConfigure = &agentsDataSource{}
)

type agentsDataSource struct {
	client *client.Client
}

type agentsDataSourceModel struct {
	Filter     types.String        `tfsdk:"filter"`
	PageSize   types.Int64         `tfsdk:"page_size"`
	OrderBy    types.String        `tfsdk:"order_by"`
	TagFilters types.Set           `tfsdk:"tag_filters"`
	Agents     []agentSummaryModel `tfsdk:"agents"`
}

type agentSummaryModel struct {
	ID               types.String `tfsdk:"id"`
	CustomerAgentID  types.String `tfsdk:"customer_agent_id"`
	DisplayName      types.String `tfsdk:"display_name"`
	ModelType        types.String `tfsdk:"model_type"`
	PhoneNumber      types.String `tfsdk:"phone_number"`
	Endpoint         types.String `tfsdk:"endpoint"`
	Prompt           types.String `tfsdk:"prompt"`
	Language         types.String `tfsdk:"language"`
	MetricIDs        types.Set    `tfsdk:"metric_ids"`
	TestSetIDs       types.Set    `tfsdk:"test_set_ids"`
	KnowledgeBaseIDs types.Set    `tfsdk:"knowledge_base_ids"`
	Tags             types.Set    `tfsdk:"tags"`
	CreateTime       types.String `tfsdk:"create_time"`
	UpdateTime       types.String `tfsdk:"update_time"`
}

func newAgentsDataSource() datasource.DataSource {
	return &agentsDataSource{}
}

func (d *agentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agents"
}

func (d *agentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all Coval agents matching public API filters.",
		Attributes: map[string]schema.Attribute{
			"filter": schema.StringAttribute{
				MarkdownDescription: "Optional public API filter expression. Quote values that contain spaces.",
				Optional:            true,
			},
			"page_size": schema.Int64Attribute{
				MarkdownDescription: "Number of agents requested per API page. The data source follows all returned pages.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.Between(1, 100)},
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Public API sort expression, such as -create_time or display_name.",
				Optional:            true,
			},
			"tag_filters": schema.SetAttribute{
				MarkdownDescription: "Tags every returned agent must contain.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators:          agentTagValidators(),
			},
			"agents": schema.ListNestedAttribute{
				MarkdownDescription: "All matching agent summaries. Use the singular coval_agent data source to retrieve free-form JSON fields.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                 schema.StringAttribute{Computed: true},
					"customer_agent_id":  schema.StringAttribute{Computed: true},
					"display_name":       schema.StringAttribute{Computed: true},
					"model_type":         schema.StringAttribute{Computed: true},
					"phone_number":       schema.StringAttribute{Computed: true},
					"endpoint":           schema.StringAttribute{Computed: true},
					"prompt":             schema.StringAttribute{Computed: true},
					"language":           schema.StringAttribute{Computed: true},
					"metric_ids":         schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"test_set_ids":       schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"knowledge_base_ids": schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"tags":               schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"create_time":        schema.StringAttribute{Computed: true},
					"update_time":        schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *agentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *agentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config agentsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tagFilters, tagDiagnostics := stringSet(ctx, config.TagFilters)
	resp.Diagnostics.Append(tagDiagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	options := client.ListAgentsOptions{
		Filter:   config.Filter.ValueString(),
		PageSize: int(config.PageSize.ValueInt64()),
		OrderBy:  config.OrderBy.ValueString(),
	}
	if tagFilters != nil {
		options.TagFilters = *tagFilters
	}
	agents, err := listAllAgents(ctx, d.client, options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval agents", err.Error())
		return
	}
	state := agentsDataSourceModel{
		Filter:     config.Filter,
		PageSize:   config.PageSize,
		OrderBy:    config.OrderBy,
		TagFilters: config.TagFilters,
		Agents:     make([]agentSummaryModel, 0, len(agents)),
	}
	for _, remote := range agents {
		metricIDs, metricDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.MetricIDs)
		resp.Diagnostics.Append(metricDiagnostics...)
		testSetIDs, testSetDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.TestSetIDs)
		resp.Diagnostics.Append(testSetDiagnostics...)
		knowledgeBaseIDs, knowledgeDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.KnowledgeBaseIDs)
		resp.Diagnostics.Append(knowledgeDiagnostics...)
		tags, tagDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.Tags)
		resp.Diagnostics.Append(tagDiagnostics...)
		state.Agents = append(state.Agents, agentSummaryModel{
			ID:               types.StringValue(remote.ID),
			CustomerAgentID:  types.StringValue(remote.CustomerAgentID),
			DisplayName:      types.StringValue(remote.DisplayName),
			ModelType:        types.StringValue(remote.ModelType),
			PhoneNumber:      nullableString(remote.PhoneNumber),
			Endpoint:         nullableString(remote.Endpoint),
			Prompt:           nullableString(remote.Prompt),
			Language:         nullableString(remote.Language),
			MetricIDs:        metricIDs,
			TestSetIDs:       testSetIDs,
			KnowledgeBaseIDs: knowledgeBaseIDs,
			Tags:             tags,
			CreateTime:       types.StringValue(remote.CreateTime),
			UpdateTime:       nullableString(remote.UpdateTime),
		})
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func listAllAgents(ctx context.Context, apiClient *client.Client, options client.ListAgentsOptions) ([]client.Agent, error) {
	var agents []client.Agent
	seenTokens := map[string]struct{}{}
	for {
		page, err := apiClient.ListAgents(ctx, options)
		if err != nil {
			return nil, err
		}
		agents = append(agents, page.Agents...)
		if page.NextPageToken == "" {
			return agents, nil
		}
		if _, exists := seenTokens[page.NextPageToken]; exists {
			return nil, fmt.Errorf("coval agents API repeated pagination token %q", page.NextPageToken)
		}
		seenTokens[page.NextPageToken] = struct{}{}
		options.PageToken = page.NextPageToken
	}
}
