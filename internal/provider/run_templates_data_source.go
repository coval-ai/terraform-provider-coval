package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &runTemplatesDataSource{}
	_ datasource.DataSourceWithConfigure = &runTemplatesDataSource{}
)

type runTemplatesDataSource struct {
	client *client.Client
}

type runTemplatesDataSourceModel struct {
	WorkspaceID  types.String              `tfsdk:"workspace_id"`
	TagFilters   types.Set                 `tfsdk:"tag_filters"`
	RunTemplates []runTemplateSummaryModel `tfsdk:"run_templates"`
}

type runTemplateSummaryModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	DisplayName     types.String `tfsdk:"display_name"`
	Description     types.String `tfsdk:"description"`
	AgentIDs        types.Set    `tfsdk:"agent_ids"`
	PersonaIDs      types.Set    `tfsdk:"persona_ids"`
	TestSetIDs      types.Set    `tfsdk:"test_set_ids"`
	MetricIDs       types.Set    `tfsdk:"metric_ids"`
	MutationIDs     types.Set    `tfsdk:"mutation_ids"`
	IterationCount  types.Int64  `tfsdk:"iteration_count"`
	Concurrency     types.Int64  `tfsdk:"concurrency"`
	SubSampleSize   types.Int64  `tfsdk:"sub_sample_size"`
	SubSampleSeed   types.Int64  `tfsdk:"sub_sample_seed"`
	Tags            types.Set    `tfsdk:"tags"`
	CreateTime      types.String `tfsdk:"create_time"`
	UpdateTime      types.String `tfsdk:"update_time"`
	CreatedByUserID types.String `tfsdk:"created_by_user_id"`
}

func newRunTemplatesDataSource() datasource.DataSource {
	return &runTemplatesDataSource{}
}

func (d *runTemplatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run_templates"
}

func (d *runTemplatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coval run templates visible to the configured API key.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": workspaceDataSourceAttribute(),
			"tag_filters": schema.SetAttribute{
				MarkdownDescription: "Optional tags that every returned run template must have.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators:          []validator.Set{setvalidator.SizeAtMost(20)},
			},
			"run_templates": schema.ListNestedAttribute{
				MarkdownDescription: "All matching run-template summaries. Use the singular coval_run_template data source to retrieve metadata.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                 schema.StringAttribute{Computed: true},
					"name":               schema.StringAttribute{Computed: true},
					"display_name":       schema.StringAttribute{Computed: true},
					"description":        schema.StringAttribute{Computed: true},
					"agent_ids":          schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"persona_ids":        schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"test_set_ids":       schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"metric_ids":         schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"mutation_ids":       schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"iteration_count":    schema.Int64Attribute{Computed: true},
					"concurrency":        schema.Int64Attribute{Computed: true},
					"sub_sample_size":    schema.Int64Attribute{Computed: true},
					"sub_sample_seed":    schema.Int64Attribute{Computed: true},
					"tags":               schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"create_time":        schema.StringAttribute{Computed: true},
					"update_time":        schema.StringAttribute{Computed: true},
					"created_by_user_id": schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *runTemplatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *runTemplatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runTemplatesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, tagDiagnostics := stringSet(ctx, config.TagFilters)
	resp.Diagnostics.Append(tagDiagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	options := client.ListRunTemplatesOptions{PageSize: 100}
	if tags != nil {
		options.TagFilters = *tags
	}
	all, err := listAllRunTemplates(ctx, clientForWorkspace(d.client, config.WorkspaceID), options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval run templates", err.Error())
		return
	}

	config.RunTemplates = make([]runTemplateSummaryModel, len(all))
	for index, remote := range all {
		state, diagnostics := runTemplateState(ctx, remote, nil)
		resp.Diagnostics.Append(diagnostics...)
		config.RunTemplates[index] = runTemplateSummaryModel{
			ID:              state.ID,
			Name:            state.Name,
			DisplayName:     state.DisplayName,
			Description:     state.Description,
			AgentIDs:        state.AgentIDs,
			PersonaIDs:      state.PersonaIDs,
			TestSetIDs:      state.TestSetIDs,
			MetricIDs:       state.MetricIDs,
			MutationIDs:     state.MutationIDs,
			IterationCount:  state.IterationCount,
			Concurrency:     state.Concurrency,
			SubSampleSize:   state.SubSampleSize,
			SubSampleSeed:   state.SubSampleSeed,
			Tags:            state.Tags,
			CreateTime:      state.CreateTime,
			UpdateTime:      state.UpdateTime,
			CreatedByUserID: state.CreatedByUserID,
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func listAllRunTemplates(ctx context.Context, apiClient *client.Client, options client.ListRunTemplatesOptions) ([]client.RunTemplate, error) {
	var all []client.RunTemplate
	seenTokens := map[string]struct{}{}
	for {
		page, err := apiClient.ListRunTemplates(ctx, options)
		if err != nil {
			return nil, err
		}
		all = append(all, page.RunTemplates...)
		if page.NextPageToken == "" {
			return all, nil
		}
		if _, duplicate := seenTokens[page.NextPageToken]; duplicate {
			return nil, fmt.Errorf("coval returned repeated pagination token %q", page.NextPageToken)
		}
		seenTokens[page.NextPageToken] = struct{}{}
		options.PageToken = page.NextPageToken
	}
}
