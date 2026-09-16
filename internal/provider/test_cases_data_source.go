package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &testCasesDataSource{}
	_ datasource.DataSourceWithConfigure = &testCasesDataSource{}
)

type testCasesDataSource struct {
	client *client.Client
}

type testCasesDataSourceModel struct {
	Filter    types.String           `tfsdk:"filter"`
	OrderBy   types.String           `tfsdk:"order_by"`
	TestCases []testCaseSummaryModel `tfsdk:"test_cases"`
}

type testCaseSummaryModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	TestSetID         types.String `tfsdk:"test_set_id"`
	InputString       types.String `tfsdk:"input_str"`
	ExpectedBehaviors types.List   `tfsdk:"expected_behaviors"`
	Description       types.String `tfsdk:"description"`
	InputType         types.String `tfsdk:"input_type"`
	UserNotes         types.String `tfsdk:"user_notes"`
	CreateTime        types.String `tfsdk:"create_time"`
	UpdateTime        types.String `tfsdk:"update_time"`
}

func newTestCasesDataSource() datasource.DataSource {
	return &testCasesDataSource{}
}

func (d *testCasesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_test_cases"
}

func (d *testCasesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coval test cases visible to the configured API key.",
		Attributes: map[string]schema.Attribute{
			"filter": schema.StringAttribute{
				MarkdownDescription: "Optional public API filter expression. Quote string values, such as test_set_id=\"abc12345\".",
				Optional:            true,
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Optional ordering expression, such as -create_time.",
				Optional:            true,
			},
			"test_cases": schema.ListNestedAttribute{
				MarkdownDescription: "All matching test-case summaries. Use the singular coval_test_case data source to retrieve dynamic JSON fields.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":                 schema.StringAttribute{Computed: true},
					"name":               schema.StringAttribute{Computed: true},
					"test_set_id":        schema.StringAttribute{Computed: true},
					"input_str":          schema.StringAttribute{Computed: true},
					"expected_behaviors": schema.ListAttribute{ElementType: types.StringType, Computed: true},
					"description":        schema.StringAttribute{Computed: true},
					"input_type":         schema.StringAttribute{Computed: true},
					"user_notes":         schema.StringAttribute{Computed: true},
					"create_time":        schema.StringAttribute{Computed: true},
					"update_time":        schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *testCasesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *testCasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config testCasesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options := client.ListTestCasesOptions{PageSize: 100}
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		options.Filter = config.Filter.ValueString()
	}
	if !config.OrderBy.IsNull() && !config.OrderBy.IsUnknown() {
		options.OrderBy = config.OrderBy.ValueString()
	}

	all, err := listAllTestCases(ctx, d.client, options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval test cases", err.Error())
		return
	}

	config.TestCases = make([]testCaseSummaryModel, len(all))
	for index, remote := range all {
		expectedBehaviors := types.ListNull(types.StringType)
		if remote.ExpectedBehaviors != nil {
			var diagnostics diag.Diagnostics
			expectedBehaviors, diagnostics = types.ListValueFrom(ctx, types.StringType, *remote.ExpectedBehaviors)
			resp.Diagnostics.Append(diagnostics...)
		}
		summary := testCaseSummaryModel{
			ID:                types.StringValue(remote.ID),
			Name:              types.StringValue(remote.Name),
			TestSetID:         types.StringNull(),
			InputString:       types.StringValue(remote.InputString),
			ExpectedBehaviors: expectedBehaviors,
			Description:       types.StringNull(),
			InputType:         types.StringNull(),
			UserNotes:         types.StringNull(),
			CreateTime:        types.StringValue(remote.CreateTime),
			UpdateTime:        types.StringNull(),
		}
		if remote.TestSetID != nil {
			summary.TestSetID = types.StringValue(*remote.TestSetID)
		}
		if remote.Description != nil {
			summary.Description = types.StringValue(*remote.Description)
		}
		if remote.InputType != nil {
			summary.InputType = types.StringValue(*remote.InputType)
		}
		if remote.UserNotes != nil {
			summary.UserNotes = types.StringValue(*remote.UserNotes)
		}
		if remote.UpdateTime != nil {
			summary.UpdateTime = types.StringValue(*remote.UpdateTime)
		}
		config.TestCases[index] = summary
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func listAllTestCases(ctx context.Context, apiClient *client.Client, options client.ListTestCasesOptions) ([]client.TestCase, error) {
	var all []client.TestCase
	seenTokens := map[string]struct{}{}
	for {
		page, err := apiClient.ListTestCases(ctx, options)
		if err != nil {
			return nil, err
		}
		all = append(all, page.TestCases...)
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
