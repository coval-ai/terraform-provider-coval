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
	_ datasource.DataSource              = &metricDataSource{}
	_ datasource.DataSourceWithConfigure = &metricDataSource{}
)

type metricDataSource struct{ client *client.Client }

func newMetricDataSource() datasource.DataSource { return &metricDataSource{} }

func (d *metricDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric"
}

func (d *metricDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	computedString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	computedFloat := func(description string) schema.Float64Attribute {
		return schema.Float64Attribute{MarkdownDescription: description, Computed: true}
	}
	computedSet := func(description string) schema.SetAttribute {
		return schema.SetAttribute{MarkdownDescription: description, ElementType: types.StringType, Computed: true}
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Coval metric by ID.", Attributes: map[string]schema.Attribute{
		"name": computedString("Canonical API resource name."), "id": schema.StringAttribute{MarkdownDescription: "Metric ID.", Required: true},
		"workspace_id": workspaceDataSourceAttribute(),
		"metric_name":  computedString("Human-readable metric name."), "description": computedString("Metric description."), "metric_type": computedString("Metric evaluation type."),
		"evaluation": metricEvaluationDataSourceSchema(), "prompt": computedString("LLM evaluation prompt."), "enabled_tools": computedSet("Enabled Agent Judge evidence tools."),
		"categories": computedSet("Classification categories."), "min_value": computedFloat("Minimum score."), "max_value": computedFloat("Maximum score."),
		"metadata_field_type": computedString("Metadata field type."), "metadata_field_key": computedString("Metadata field key."), "regex_pattern": computedString("Transcript regex pattern."),
		"role": computedString("Canonical speaker role."), "min_pause_duration_seconds": computedFloat("Minimum pause duration."), "max_silence_duration_seconds": computedFloat("Maximum silence duration."),
		"min_silence_gap_seconds": computedFloat("Minimum silence gap."), "frequency_threshold": computedFloat("Audio frequency threshold."), "direction": computedString("Threshold direction."),
		"success_sentiments": computedSet("Successful sentiments."), "percent_above": computedFloat("Required percentage above threshold."), "success_end_reasons": computedSet("Successful end reasons."),
		"observation_name": computedString("Observation name."), "expected_body": schema.DynamicAttribute{MarkdownDescription: "Expected response body.", Computed: true},
		"match_path": computedString("Expected-body match path."), "min_volume_change_for_pitch_misalignment": computedFloat("Minimum volume change for pitch misalignment."),
		"threshold": schema.Int64Attribute{MarkdownDescription: "Integer threshold.", Computed: true}, "operator": computedString("Threshold comparison operator."),
		"ivr_flow": schema.DynamicAttribute{MarkdownDescription: "IVR flow JSON object.", Computed: true}, "sql_query": computedString("SQL metric query."),
		"aggregation_method": computedString("Aggregation method for custom trace or SQL float values."), "unit": computedString("Display unit for custom trace or SQL float values."),
		"criteria_source": computedString("Composite criteria source."), "criteria_path": computedString("Composite criteria path."), "criteria": computedSet("Composite literal criteria."),
		"reporting_method": computedString("Composite reporting method."), "base_prompt_template": computedString("Composite prompt template."),
		"include_traces": schema.BoolAttribute{MarkdownDescription: "Whether trace context is included.", Computed: true},
		"runtime_config": metricRuntimeConfigDataSourceSchema(), "target_condition": metricTargetConditionDataSourceSchema(), "tags": computedSet("Metric tags."),
		"created_by": computedString("Creator email returned by Coval."), "create_time": computedString("RFC 3339 creation timestamp."), "update_time": computedString("RFC 3339 update timestamp."),
		"current_version": currentMetricVersionDataSourceSchema(),
	}}
}

func metricRuntimeConfigDataSourceSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Computed: true, Attributes: map[string]schema.Attribute{"model_version": schema.StringAttribute{Computed: true}, "thinking_enabled": schema.BoolAttribute{Computed: true}}}
}
func metricTargetConditionDataSourceSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Computed: true, Attributes: map[string]schema.Attribute{"comparison_operator": schema.StringAttribute{Computed: true}, "target_float": schema.Float64Attribute{Computed: true}, "target_values": schema.SetAttribute{ElementType: types.StringType, Computed: true}}}
}
func metricEvaluationDataSourceSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Computed: true, Attributes: map[string]schema.Attribute{"evaluator": schema.StringAttribute{Computed: true}, "output_type": schema.StringAttribute{Computed: true}, "output_type_source": schema.StringAttribute{Computed: true}, "semantic_type": schema.StringAttribute{Computed: true}, "semantic_type_source": schema.StringAttribute{Computed: true}}}
}
func currentMetricVersionDataSourceSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{Computed: true, Attributes: map[string]schema.Attribute{"ulid": schema.StringAttribute{Computed: true}, "version_number": schema.Int64Attribute{Computed: true}, "change_type": schema.StringAttribute{Computed: true}}}
}

func (d *metricDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *metricDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config metricResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := clientForWorkspace(d.client, config.WorkspaceID).GetMetric(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval metric", err.Error())
		return
	}
	state, diagnostics := metricState(ctx, remote, &config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
