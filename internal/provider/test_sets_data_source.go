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
	_ datasource.DataSource              = &testSetsDataSource{}
	_ datasource.DataSourceWithConfigure = &testSetsDataSource{}
)

type testSetsDataSource struct {
	client *client.Client
}

type testSetsDataSourceModel struct {
	Filter     types.String          `tfsdk:"filter"`
	OrderBy    types.String          `tfsdk:"order_by"`
	TagFilters types.Set             `tfsdk:"tag_filters"`
	TestSets   []testSetSummaryModel `tfsdk:"test_sets"`
}

type testSetSummaryModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Slug          types.String `tfsdk:"slug"`
	DisplayName   types.String `tfsdk:"display_name"`
	Description   types.String `tfsdk:"description"`
	TestSetType   types.String `tfsdk:"test_set_type"`
	TestCaseCount types.Int64  `tfsdk:"test_case_count"`
	Tags          types.Set    `tfsdk:"tags"`
	CreateTime    types.String `tfsdk:"create_time"`
	UpdateTime    types.String `tfsdk:"update_time"`
}

func newTestSetsDataSource() datasource.DataSource {
	return &testSetsDataSource{}
}

func (d *testSetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_test_sets"
}

func (d *testSetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coval test sets visible to the configured API key.",
		Attributes: map[string]schema.Attribute{
			"filter": schema.StringAttribute{
				MarkdownDescription: "Optional public API filter expression. Quote string values.",
				Optional:            true,
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Optional ordering expression, such as -create_time.",
				Optional:            true,
			},
			"tag_filters": schema.SetAttribute{
				MarkdownDescription: "Optional tags that every returned test set must have.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators:          []validator.Set{setvalidator.SizeAtMost(20)},
			},
			"test_sets": schema.ListNestedAttribute{
				MarkdownDescription: "All matching test sets.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":              schema.StringAttribute{Computed: true},
					"name":            schema.StringAttribute{Computed: true},
					"slug":            schema.StringAttribute{Computed: true},
					"display_name":    schema.StringAttribute{Computed: true},
					"description":     schema.StringAttribute{Computed: true},
					"test_set_type":   schema.StringAttribute{Computed: true},
					"test_case_count": schema.Int64Attribute{Computed: true},
					"tags":            schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"create_time":     schema.StringAttribute{Computed: true},
					"update_time":     schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *testSetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *testSetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config testSetsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, tagDiagnostics := stringSet(ctx, config.TagFilters)
	resp.Diagnostics.Append(tagDiagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	options := client.ListTestSetsOptions{PageSize: 100}
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		options.Filter = config.Filter.ValueString()
	}
	if !config.OrderBy.IsNull() && !config.OrderBy.IsUnknown() {
		options.OrderBy = config.OrderBy.ValueString()
	}
	if tags != nil {
		options.TagFilters = *tags
	}

	all, err := listAllTestSets(ctx, d.client, options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval test sets", err.Error())
		return
	}

	config.TestSets = make([]testSetSummaryModel, len(all))
	for index, remote := range all {
		tagSet, diagnostics := types.SetValueFrom(ctx, types.StringType, remote.Tags)
		resp.Diagnostics.Append(diagnostics...)
		summary := testSetSummaryModel{
			ID:            types.StringValue(remote.ID),
			Name:          types.StringValue(remote.Name),
			Slug:          types.StringValue(remote.Slug),
			DisplayName:   types.StringValue(remote.DisplayName),
			Description:   types.StringNull(),
			TestSetType:   types.StringNull(),
			TestCaseCount: types.Int64Null(),
			Tags:          tagSet,
			CreateTime:    types.StringValue(remote.CreateTime),
			UpdateTime:    types.StringNull(),
		}
		if remote.Description != nil {
			summary.Description = types.StringValue(*remote.Description)
		}
		if remote.TestSetType != nil {
			summary.TestSetType = types.StringValue(*remote.TestSetType)
		}
		if remote.TestCaseCount != nil {
			summary.TestCaseCount = types.Int64Value(*remote.TestCaseCount)
		}
		if remote.UpdateTime != nil {
			summary.UpdateTime = types.StringValue(*remote.UpdateTime)
		}
		config.TestSets[index] = summary
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func listAllTestSets(ctx context.Context, apiClient *client.Client, options client.ListTestSetsOptions) ([]client.TestSet, error) {
	var all []client.TestSet
	seenTokens := map[string]struct{}{}
	for {
		page, err := apiClient.ListTestSets(ctx, options)
		if err != nil {
			return nil, err
		}
		all = append(all, page.TestSets...)
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
