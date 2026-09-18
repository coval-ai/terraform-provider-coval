package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &metricsDataSource{}
	_ datasource.DataSourceWithConfigure = &metricsDataSource{}
)

type metricsDataSource struct{ client *client.Client }
type metricsDataSourceModel struct {
	Filter         types.String         `tfsdk:"filter"`
	PageSize       types.Int64          `tfsdk:"page_size"`
	OrderBy        types.String         `tfsdk:"order_by"`
	IncludeBuiltin types.Bool           `tfsdk:"include_builtin"`
	TagFilters     types.Set            `tfsdk:"tag_filters"`
	Metrics        []metricSummaryModel `tfsdk:"metrics"`
}
type metricSummaryModel struct {
	Name                                types.String  `tfsdk:"name"`
	ID                                  types.String  `tfsdk:"id"`
	MetricName                          types.String  `tfsdk:"metric_name"`
	Description                         types.String  `tfsdk:"description"`
	MetricType                          types.String  `tfsdk:"metric_type"`
	Prompt                              types.String  `tfsdk:"prompt"`
	EnabledTools                        types.Set     `tfsdk:"enabled_tools"`
	Categories                          types.Set     `tfsdk:"categories"`
	MinValue                            types.Float64 `tfsdk:"min_value"`
	MaxValue                            types.Float64 `tfsdk:"max_value"`
	MetadataFieldType                   types.String  `tfsdk:"metadata_field_type"`
	MetadataFieldKey                    types.String  `tfsdk:"metadata_field_key"`
	RegexPattern                        types.String  `tfsdk:"regex_pattern"`
	Role                                types.String  `tfsdk:"role"`
	MinPauseDurationSeconds             types.Float64 `tfsdk:"min_pause_duration_seconds"`
	MaxSilenceDurationSeconds           types.Float64 `tfsdk:"max_silence_duration_seconds"`
	MinSilenceGapSeconds                types.Float64 `tfsdk:"min_silence_gap_seconds"`
	FrequencyThreshold                  types.Float64 `tfsdk:"frequency_threshold"`
	Direction                           types.String  `tfsdk:"direction"`
	SuccessSentiments                   types.Set     `tfsdk:"success_sentiments"`
	PercentAbove                        types.Float64 `tfsdk:"percent_above"`
	SuccessEndReasons                   types.Set     `tfsdk:"success_end_reasons"`
	ObservationName                     types.String  `tfsdk:"observation_name"`
	MatchPath                           types.String  `tfsdk:"match_path"`
	MinVolumeChangeForPitchMisalignment types.Float64 `tfsdk:"min_volume_change_for_pitch_misalignment"`
	Threshold                           types.Int64   `tfsdk:"threshold"`
	Operator                            types.String  `tfsdk:"operator"`
	SQLQuery                            types.String  `tfsdk:"sql_query"`
	CriteriaSource                      types.String  `tfsdk:"criteria_source"`
	CriteriaPath                        types.String  `tfsdk:"criteria_path"`
	Criteria                            types.Set     `tfsdk:"criteria"`
	ReportingMethod                     types.String  `tfsdk:"reporting_method"`
	BasePromptTemplate                  types.String  `tfsdk:"base_prompt_template"`
	IncludeTraces                       types.Bool    `tfsdk:"include_traces"`
	Tags                                types.Set     `tfsdk:"tags"`
	CreatedBy                           types.String  `tfsdk:"created_by"`
	CreateTime                          types.String  `tfsdk:"create_time"`
	UpdateTime                          types.String  `tfsdk:"update_time"`
}

func newMetricsDataSource() datasource.DataSource { return &metricsDataSource{} }
func (d *metricsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metrics"
}

func (d *metricsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	cS := func() schema.StringAttribute { return schema.StringAttribute{Computed: true} }
	cF := func() schema.Float64Attribute { return schema.Float64Attribute{Computed: true} }
	cSet := func() schema.SetAttribute { return schema.SetAttribute{ElementType: types.StringType, Computed: true} }
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves all Coval metrics matching public API filters.", Attributes: map[string]schema.Attribute{
		"filter": schema.StringAttribute{Optional: true}, "page_size": schema.Int64Attribute{Optional: true, Validators: []validator.Int64{int64validator.Between(1, 100)}}, "order_by": schema.StringAttribute{Optional: true},
		"include_builtin": schema.BoolAttribute{MarkdownDescription: "Include Coval built-in metrics.", Optional: true}, "tag_filters": schema.SetAttribute{ElementType: types.StringType, Optional: true, Validators: []validator.Set{setvalidator.SizeAtMost(20)}},
		"metrics": schema.ListNestedAttribute{MarkdownDescription: "All matching metric summaries. Use the singular coval_metric data source for polymorphic and nested configuration.", Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"name": cS(), "id": cS(), "metric_name": cS(), "description": cS(), "metric_type": cS(), "prompt": cS(), "enabled_tools": cSet(), "categories": cSet(), "min_value": cF(), "max_value": cF(),
			"metadata_field_type": cS(), "metadata_field_key": cS(), "regex_pattern": cS(), "role": cS(), "min_pause_duration_seconds": cF(), "max_silence_duration_seconds": cF(), "min_silence_gap_seconds": cF(), "frequency_threshold": cF(),
			"direction": cS(), "success_sentiments": cSet(), "percent_above": cF(), "success_end_reasons": cSet(), "observation_name": cS(), "match_path": cS(), "min_volume_change_for_pitch_misalignment": cF(), "threshold": schema.Int64Attribute{Computed: true}, "operator": cS(), "sql_query": cS(),
			"criteria_source": cS(), "criteria_path": cS(), "criteria": cSet(), "reporting_method": cS(), "base_prompt_template": cS(), "include_traces": schema.BoolAttribute{Computed: true}, "tags": cSet(), "created_by": cS(), "create_time": cS(), "update_time": cS(),
		}}},
	}}
}

func (d *metricsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *metricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config metricsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tags, tagDiagnostics := stringSet(ctx, config.TagFilters)
	resp.Diagnostics.Append(tagDiagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	options := client.ListMetricsOptions{Filter: config.Filter.ValueString(), PageSize: int(config.PageSize.ValueInt64()), OrderBy: config.OrderBy.ValueString(), IncludeBuiltin: config.IncludeBuiltin.ValueBool()}
	if tags != nil {
		options.TagFilters = *tags
	}
	all, err := listAllMetrics(ctx, d.client, options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval metrics", err.Error())
		return
	}
	config.Metrics = make([]metricSummaryModel, len(all))
	for index, remote := range all {
		set := func(values *[]string) types.Set {
			if values == nil {
				return types.SetNull(types.StringType)
			}
			result, diagnostics := types.SetValueFrom(ctx, types.StringType, *values)
			resp.Diagnostics.Append(diagnostics...)
			return result
		}
		tags, diagnostics := types.SetValueFrom(ctx, types.StringType, remote.Tags)
		resp.Diagnostics.Append(diagnostics...)
		config.Metrics[index] = metricSummaryModel{Name: types.StringValue(remote.Name), ID: types.StringValue(remote.ID), MetricName: types.StringValue(remote.MetricName), Description: types.StringValue(remote.Description), MetricType: types.StringValue(remote.MetricType), Prompt: nullableMetricString(remote.Prompt), EnabledTools: set(remote.EnabledTools), Categories: set(remote.Categories), MinValue: nullableFloat(remote.MinValue), MaxValue: nullableFloat(remote.MaxValue), MetadataFieldType: nullableMetricString(remote.MetadataFieldType), MetadataFieldKey: nullableMetricString(remote.MetadataFieldKey), RegexPattern: nullableMetricString(remote.RegexPattern), Role: nullableMetricString(remote.Role), MinPauseDurationSeconds: nullableFloat(remote.MinPauseDurationSeconds), MaxSilenceDurationSeconds: nullableFloat(remote.MaxSilenceDurationSeconds), MinSilenceGapSeconds: nullableFloat(remote.MinSilenceGapSeconds), FrequencyThreshold: nullableFloat(remote.FrequencyThreshold), Direction: nullableMetricString(remote.Direction), SuccessSentiments: set(remote.SuccessSentiments), PercentAbove: nullableFloat(remote.PercentAbove), SuccessEndReasons: set(remote.SuccessEndReasons), ObservationName: nullableMetricString(remote.ObservationName), MatchPath: nullableMetricString(remote.MatchPath), MinVolumeChangeForPitchMisalignment: nullableFloat(remote.MinVolumeChangeForPitchMisalignment), Threshold: nullableInt(remote.Threshold), Operator: nullableMetricString(remote.Operator), SQLQuery: nullableMetricString(remote.SQLQuery), CriteriaSource: nullableMetricString(remote.CriteriaSource), CriteriaPath: nullableMetricString(remote.CriteriaPath), Criteria: set(remote.Criteria), ReportingMethod: nullableMetricString(remote.ReportingMethod), BasePromptTemplate: nullableMetricString(remote.BasePromptTemplate), IncludeTraces: nullableBool(remote.IncludeTraces), Tags: tags, CreatedBy: nullableMetricString(remote.CreatedBy), CreateTime: types.StringValue(remote.CreateTime), UpdateTime: nullableMetricString(remote.UpdateTime)}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func listAllMetrics(ctx context.Context, apiClient *client.Client, options client.ListMetricsOptions) ([]client.Metric, error) {
	var all []client.Metric
	seenTokens := map[string]struct{}{}
	for {
		page, err := apiClient.ListMetrics(ctx, options)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Metrics...)
		if page.NextPageToken == "" {
			return all, nil
		}
		if _, duplicate := seenTokens[page.NextPageToken]; duplicate {
			return nil, fmt.Errorf("coval metrics API repeated pagination token %q", page.NextPageToken)
		}
		seenTokens[page.NextPageToken] = struct{}{}
		options.PageToken = page.NextPageToken
	}
}
