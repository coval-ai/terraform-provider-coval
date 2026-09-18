package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &metricResource{}
	_ resource.ResourceWithConfigure   = &metricResource{}
	_ resource.ResourceWithImportState = &metricResource{}
	_ resource.ResourceWithIdentity    = &metricResource{}
)

var metricTypes = []string{
	"METRIC_AGENT_JUDGE", "METRIC_LLM_BINARY", "METRIC_CATEGORICAL", "METRIC_NUMERICAL_LLM_JUDGE",
	"METRIC_AUDIO_LLM_BINARY", "METRIC_AUDIO_LLM_CATEGORICAL", "METRIC_AUDIO_LLM_NUMERICAL",
	"METRIC_TOOLCALL", "METRIC_METADATA_FIELD", "METRIC_TRANSCRIPT_REGEX", "METRIC_PAUSE_ANALYSIS",
	"METRIC_SQL_FLOAT", "METRIC_COMPOSITE_EVALUATION", "METRIC_CUSTOM_AGENT_FAILS_TO_RESPOND",
	"METRIC_CUSTOM_AGENT_NEEDS_REPROMPTING", "METRIC_CUSTOM_AUDIO_FREQUENCY", "METRIC_CUSTOM_AUDIO_SENTIMENT",
	"METRIC_CUSTOM_END_REASON", "METRIC_MATCH_EXPECTED_OUTPUT", "METRIC_SPEAKING_TIME_PERCENTAGE",
	"METRIC_SPECTROGRAM_PITCH_ANALYSIS", "METRIC_VOLUME_PITCH_MISALIGNMENT", "METRIC_WORDS_PER_MESSAGE_WITH_THRESHOLD",
}

var metricRuntimeConfigAttributeTypes = map[string]attr.Type{
	"model_version":    types.StringType,
	"thinking_enabled": types.BoolType,
}

var metricTargetConditionAttributeTypes = map[string]attr.Type{
	"comparison_operator": types.StringType,
	"target_float":        types.Float64Type,
	"target_values":       types.SetType{ElemType: types.StringType},
}

var metricEvaluationAttributeTypes = map[string]attr.Type{
	"evaluator":            types.StringType,
	"output_type":          types.StringType,
	"output_type_source":   types.StringType,
	"semantic_type":        types.StringType,
	"semantic_type_source": types.StringType,
}

var currentMetricVersionAttributeTypes = map[string]attr.Type{
	"ulid":           types.StringType,
	"version_number": types.Int64Type,
	"change_type":    types.StringType,
}

type metricResource struct {
	client *client.Client
}

type metricResourceModel struct {
	Name                                types.String  `tfsdk:"name"`
	ID                                  types.String  `tfsdk:"id"`
	MetricName                          types.String  `tfsdk:"metric_name"`
	Description                         types.String  `tfsdk:"description"`
	MetricType                          types.String  `tfsdk:"metric_type"`
	Evaluation                          types.Object  `tfsdk:"evaluation"`
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
	ExpectedBody                        types.Dynamic `tfsdk:"expected_body"`
	MatchPath                           types.String  `tfsdk:"match_path"`
	MinVolumeChangeForPitchMisalignment types.Float64 `tfsdk:"min_volume_change_for_pitch_misalignment"`
	Threshold                           types.Int64   `tfsdk:"threshold"`
	Operator                            types.String  `tfsdk:"operator"`
	IVRFlow                             types.Dynamic `tfsdk:"ivr_flow"`
	SQLQuery                            types.String  `tfsdk:"sql_query"`
	CriteriaSource                      types.String  `tfsdk:"criteria_source"`
	CriteriaPath                        types.String  `tfsdk:"criteria_path"`
	Criteria                            types.Set     `tfsdk:"criteria"`
	ReportingMethod                     types.String  `tfsdk:"reporting_method"`
	BasePromptTemplate                  types.String  `tfsdk:"base_prompt_template"`
	IncludeTraces                       types.Bool    `tfsdk:"include_traces"`
	RuntimeConfig                       types.Object  `tfsdk:"runtime_config"`
	TargetCondition                     types.Object  `tfsdk:"target_condition"`
	Tags                                types.Set     `tfsdk:"tags"`
	CreatedBy                           types.String  `tfsdk:"created_by"`
	CreateTime                          types.String  `tfsdk:"create_time"`
	UpdateTime                          types.String  `tfsdk:"update_time"`
	CurrentVersion                      types.Object  `tfsdk:"current_version"`
}

type metricIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func newMetricResource() resource.Resource { return &metricResource{} }

func (r *metricResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metric"
}

func (r *metricResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optionalString := func(description string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{MarkdownDescription: description, Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}, Validators: validators}
	}
	optionalFloat := func(description string, validators ...validator.Float64) schema.Float64Attribute {
		return schema.Float64Attribute{MarkdownDescription: description, Optional: true, Computed: true, PlanModifiers: []planmodifier.Float64{float64planmodifier.UseStateForUnknown()}, Validators: validators}
	}
	optionalSet := func(description string, validators ...validator.Set) schema.SetAttribute {
		return schema.SetAttribute{MarkdownDescription: description, ElementType: types.StringType, Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}, Validators: validators}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coval evaluation metric.",
		Attributes: map[string]schema.Attribute{
			"name":                         schema.StringAttribute{MarkdownDescription: "Canonical API resource name.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"id":                           schema.StringAttribute{MarkdownDescription: "Server-assigned metric ID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"metric_name":                  schema.StringAttribute{MarkdownDescription: "Human-readable metric name.", Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 200)}},
			"description":                  schema.StringAttribute{MarkdownDescription: "Metric description.", Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 1000)}},
			"metric_type":                  schema.StringAttribute{MarkdownDescription: "Metric evaluation type.", Required: true, Validators: []validator.String{stringvalidator.OneOf(metricTypes...)}},
			"evaluation":                   metricEvaluationSchema(),
			"prompt":                       optionalString("LLM evaluation prompt."),
			"enabled_tools":                optionalSet("Evidence tools available to an Agent Judge metric. Set [] to disable every tool.", setvalidator.ValueStringsAre(stringvalidator.OneOf("get_transcript", "search_transcript", "search_transcript_regex", "get_trace_spans", "query_simulation_frames", "get_run_context"))),
			"categories":                   optionalSet("Classification categories for categorical metrics.", setvalidator.SizeBetween(2, 50)),
			"min_value":                    optionalFloat("Minimum score for numerical metrics."),
			"max_value":                    optionalFloat("Maximum score for numerical metrics."),
			"metadata_field_type":          optionalString("Data type for metadata-field extraction.", stringvalidator.OneOf("STRING", "NUMBER", "BOOLEAN")),
			"metadata_field_key":           optionalString("Metadata key extracted by a metadata-field metric."),
			"regex_pattern":                optionalString("Regular expression used by a transcript-regex metric."),
			"role":                         optionalString("Speaker role filtered by a transcript-regex metric. The API normalizes user to persona and assistant to agent.", stringvalidator.OneOf("agent", "persona", "user", "assistant")),
			"min_pause_duration_seconds":   optionalFloat("Minimum pause duration for pause-analysis metrics.", float64validator.AtLeast(0.5)),
			"max_silence_duration_seconds": optionalFloat("Maximum silence duration in seconds.", float64validator.AtLeast(math.SmallestNonzeroFloat64)),
			"min_silence_gap_seconds":      optionalFloat("Minimum gap between silence periods in seconds.", float64validator.AtLeast(math.SmallestNonzeroFloat64)),
			"frequency_threshold":          optionalFloat("Audio frequency threshold.", float64validator.AtLeast(math.SmallestNonzeroFloat64)),
			"direction":                    optionalString("Threshold direction.", stringvalidator.OneOf("above", "below")),
			"success_sentiments":           optionalSet("Sentiments that count as successful.", setvalidator.SizeAtLeast(1), setvalidator.ValueStringsAre(stringvalidator.OneOf("Neutral", "Happy", "Angry", "Sad"))),
			"percent_above":                optionalFloat("Required percentage above the threshold.", float64validator.Between(0, 100)),
			"success_end_reasons":          optionalSet("Simulation end reasons that count as successful.", setvalidator.SizeAtLeast(1), setvalidator.ValueStringsAre(stringvalidator.OneOf("UNKNOWN", "IDLE_TIMEOUT", "DURATION_LIMIT", "PERSONA_DISCONNECTED", "AGENT_DISCONNECTED", "PIPELINE_ERROR", "REPETITION_LOOP", "AUDIO_UPLOAD_PLAYBACK_COMPLETED", "SCRIPT_COMPLETED", "SCRIPT_DIVERGED"))),
			"observation_name":             optionalString("Observation name evaluated by the metric.", stringvalidator.LengthAtLeast(1)),
			"expected_body":                schema.DynamicAttribute{MarkdownDescription: "Expected response body as a non-empty string or JSON object.", Optional: true, Computed: true, PlanModifiers: []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()}},
			"match_path":                   optionalString("Optional dot path used to select a value from the expected body.", stringvalidator.LengthAtLeast(1)),
			"min_volume_change_for_pitch_misalignment": optionalFloat("Minimum volume change for pitch-misalignment detection.", float64validator.AtLeast(math.SmallestNonzeroFloat64)),
			"threshold":            schema.Int64Attribute{MarkdownDescription: "Integer threshold used by threshold-based metrics.", Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Validators: []validator.Int64{int64validator.AtLeast(0)}},
			"operator":             optionalString("Comparison operator used with threshold.", stringvalidator.OneOf("<", "<=", ">", ">=", "==", "!=")),
			"ivr_flow":             schema.DynamicAttribute{MarkdownDescription: "IVR flow JSON object for IVR flow-adherence metrics.", Optional: true, Computed: true, PlanModifiers: []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()}},
			"sql_query":            optionalString("SQL query used by a SQL float metric.", stringvalidator.LengthAtMost(50000)),
			"criteria_source":      optionalString("Source used by a composite-evaluation metric.", stringvalidator.OneOf("test_case", "test_case_attribute", "metric_metadata")),
			"criteria_path":        optionalString("Path to criteria on the selected source.", stringvalidator.LengthAtMost(200)),
			"criteria":             optionalSet("Literal criteria for a composite-evaluation metric."),
			"reporting_method":     optionalString("How composite criterion verdicts are aggregated.", stringvalidator.OneOf("percentage_of_criteria_met", "count_of_criteria_met", "all_criteria_met")),
			"base_prompt_template": optionalString("Custom per-criterion evaluation prompt template.", stringvalidator.LengthAtMost(50000)),
			"include_traces":       schema.BoolAttribute{MarkdownDescription: "Whether trace context is injected into supported LLM judge prompts.", Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
			"runtime_config":       metricRuntimeConfigSchema(),
			"target_condition":     metricTargetConditionSchema(),
			"tags":                 optionalSet("Tags associated with the metric. Set [] to clear them."),
			"created_by":           schema.StringAttribute{MarkdownDescription: "Creator email returned by Coval.", Computed: true},
			"create_time":          schema.StringAttribute{MarkdownDescription: "RFC 3339 creation timestamp.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"update_time":          schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest update.", Computed: true},
			"current_version":      currentMetricVersionSchema(),
		},
	}
}

func metricRuntimeConfigSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "LLM model and thinking configuration. Set an empty object to restore the platform default during an update.",
		Optional:            true, Computed: true,
		PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		Attributes: map[string]schema.Attribute{
			"model_version":    schema.StringAttribute{Optional: true, Computed: true},
			"thinking_enabled": schema.BoolAttribute{Optional: true, Computed: true},
		},
	}
}

func metricTargetConditionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Rule that determines which metric output counts as a success.",
		Optional:            true, Computed: true,
		PlanModifiers: []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		Attributes: map[string]schema.Attribute{
			"comparison_operator": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("eq", "neq", "gt", "gte", "lt", "lte", "in", "nin")}},
			"target_float":        schema.Float64Attribute{Optional: true},
			"target_values":       schema.SetAttribute{ElementType: types.StringType, Optional: true, Validators: []validator.Set{setvalidator.SizeBetween(1, 50)}},
		},
	}
}

func metricEvaluationSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{MarkdownDescription: "Read-only evaluator catalog metadata.", Computed: true, Attributes: map[string]schema.Attribute{
		"evaluator": schema.StringAttribute{Computed: true}, "output_type": schema.StringAttribute{Computed: true},
		"output_type_source": schema.StringAttribute{Computed: true}, "semantic_type": schema.StringAttribute{Computed: true},
		"semantic_type_source": schema.StringAttribute{Computed: true},
	}}
}

func currentMetricVersionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{MarkdownDescription: "Current live metric version.", Computed: true, Attributes: map[string]schema.Attribute{
		"ulid": schema.StringAttribute{Computed: true}, "version_number": schema.Int64Attribute{Computed: true}, "change_type": schema.StringAttribute{Computed: true},
	}}
}

func (r *metricResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{"id": identityschema.StringAttribute{RequiredForImport: true}}}
}

func (r *metricResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return
	}
	r.client = apiClient
}

func (r *metricResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan metricResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, diagnostics := metricInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateMetric(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval metric", err.Error())
		return
	}
	state, diagnostics := metricResourceState(ctx, created, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, metricIdentityModel{ID: state.ID})...)
}

func (r *metricResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state metricResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.client.GetMetric(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval metric", err.Error())
		return
	}
	refreshed, diagnostics := metricResourceState(ctx, remote, &state)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, metricIdentityModel{ID: refreshed.ID})...)
}

func (r *metricResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan metricResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var config metricResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, diagnostics := metricInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateMetric(ctx, plan.ID.ValueString(), client.UpdateMetricInput{
		CreateMetricInput:  input,
		ClearRuntimeConfig: runtimeConfigIsExplicitlyEmpty(config.RuntimeConfig),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval metric", err.Error())
		return
	}
	state, diagnostics := metricResourceState(ctx, updated, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, metricIdentityModel{ID: state.ID})...)
}

func (r *metricResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state metricResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMetric(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval metric", err.Error())
	}
}

func (r *metricResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func metricInput(ctx context.Context, plan metricResourceModel) (client.CreateMetricInput, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	set := func(value types.Set) *[]string {
		result, d := stringSet(ctx, value)
		diagnostics.Append(d...)
		return result
	}
	expectedBody, err := metricExpectedBody(plan.ExpectedBody)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("expected_body"), "Invalid expected body", err.Error())
	}
	ivrFlow, err := dynamicJSONObject(plan.IVRFlow)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("ivr_flow"), "Invalid IVR flow", err.Error())
	}
	runtimeConfig, d := runtimeConfigFromObject(ctx, plan.RuntimeConfig)
	diagnostics.Append(d...)
	targetCondition, d := targetConditionFromObject(ctx, plan.TargetCondition)
	diagnostics.Append(d...)
	return client.CreateMetricInput{
		MetricName: plan.MetricName.ValueString(), Description: plan.Description.ValueString(), MetricType: plan.MetricType.ValueString(),
		Prompt: stringPointer(plan.Prompt), EnabledTools: set(plan.EnabledTools), Categories: set(plan.Categories),
		MinValue: floatPointer(plan.MinValue), MaxValue: floatPointer(plan.MaxValue), MetadataFieldType: stringPointer(plan.MetadataFieldType),
		MetadataFieldKey: stringPointer(plan.MetadataFieldKey), RegexPattern: stringPointer(plan.RegexPattern), Role: stringPointer(plan.Role),
		MinPauseDurationSeconds: floatPointer(plan.MinPauseDurationSeconds), MaxSilenceDurationSeconds: floatPointer(plan.MaxSilenceDurationSeconds),
		MinSilenceGapSeconds: floatPointer(plan.MinSilenceGapSeconds), FrequencyThreshold: floatPointer(plan.FrequencyThreshold),
		Direction: stringPointer(plan.Direction), SuccessSentiments: set(plan.SuccessSentiments), PercentAbove: floatPointer(plan.PercentAbove),
		SuccessEndReasons: set(plan.SuccessEndReasons), ObservationName: stringPointer(plan.ObservationName), ExpectedBody: expectedBody,
		MatchPath: stringPointer(plan.MatchPath), MinVolumeChangeForPitchMisalignment: floatPointer(plan.MinVolumeChangeForPitchMisalignment),
		Threshold: intPointer(plan.Threshold), Operator: stringPointer(plan.Operator), IVRFlow: ivrFlow, SQLQuery: stringPointer(plan.SQLQuery),
		CriteriaSource: stringPointer(plan.CriteriaSource), CriteriaPath: stringPointer(plan.CriteriaPath), Criteria: set(plan.Criteria),
		ReportingMethod: stringPointer(plan.ReportingMethod), BasePromptTemplate: stringPointer(plan.BasePromptTemplate),
		IncludeTraces: boolPointer(plan.IncludeTraces), RuntimeConfig: runtimeConfig, TargetCondition: targetCondition, Tags: set(plan.Tags),
	}, diagnostics
}

func metricExpectedBody(value types.Dynamic) (*json.RawMessage, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	if value.IsUnderlyingValueUnknown() {
		return nil, fmt.Errorf("value must be known before it can be sent to Coval")
	}
	switch typed := value.UnderlyingValue().(type) {
	case types.String:
		if typed.ValueString() == "" {
			return nil, fmt.Errorf("string value must not be empty")
		}
	case types.Map:
		if len(typed.Elements()) == 0 {
			return nil, fmt.Errorf("object value must not be empty")
		}
	case types.Object:
		if len(typed.Attributes()) == 0 {
			return nil, fmt.Errorf("object value must not be empty")
		}
	default:
		return nil, fmt.Errorf("value must be a string or object, got %T", typed)
	}
	return dynamicJSONValue(value)
}

func floatPointer(value types.Float64) *float64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueFloat64()
	return &result
}
func intPointer(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueInt64()
	return &result
}
func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}

func runtimeConfigFromObject(_ context.Context, value types.Object) (*client.MetricRuntimeConfig, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var diagnostics diag.Diagnostics
	attributes := value.Attributes()
	modelVersion, ok := attributes["model_version"].(types.String)
	if !ok {
		diagnostics.AddAttributeError(path.Root("runtime_config").AtName("model_version"), "Invalid runtime configuration", fmt.Sprintf("Expected a string value, got %T.", attributes["model_version"]))
	}
	thinkingEnabled, ok := attributes["thinking_enabled"].(types.Bool)
	if !ok {
		diagnostics.AddAttributeError(path.Root("runtime_config").AtName("thinking_enabled"), "Invalid runtime configuration", fmt.Sprintf("Expected a boolean value, got %T.", attributes["thinking_enabled"]))
	}
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	return &client.MetricRuntimeConfig{ModelVersion: stringPointer(modelVersion), ThinkingEnabled: boolPointer(thinkingEnabled)}, diagnostics
}

func runtimeConfigIsExplicitlyEmpty(value types.Object) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	attributes := value.Attributes()
	return attributes["model_version"].IsNull() && attributes["thinking_enabled"].IsNull()
}

func targetConditionFromObject(ctx context.Context, value types.Object) (*client.MetricTargetCondition, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var diagnostics diag.Diagnostics
	attributes := value.Attributes()
	comparisonOperator, ok := attributes["comparison_operator"].(types.String)
	if !ok {
		diagnostics.AddAttributeError(path.Root("target_condition").AtName("comparison_operator"), "Invalid target condition", fmt.Sprintf("Expected a string value, got %T.", attributes["comparison_operator"]))
	}
	targetFloatValue, ok := attributes["target_float"].(types.Float64)
	if !ok {
		diagnostics.AddAttributeError(path.Root("target_condition").AtName("target_float"), "Invalid target condition", fmt.Sprintf("Expected a number value, got %T.", attributes["target_float"]))
	}
	targetValuesValue, ok := attributes["target_values"].(types.Set)
	if !ok {
		diagnostics.AddAttributeError(path.Root("target_condition").AtName("target_values"), "Invalid target condition", fmt.Sprintf("Expected a set value, got %T.", attributes["target_values"]))
	}
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	targetFloat := floatPointer(targetFloatValue)
	targetValues, setDiagnostics := stringSet(ctx, targetValuesValue)
	diagnostics.Append(setDiagnostics...)
	if comparisonOperator.IsUnknown() {
		return nil, diagnostics
	}
	hasFloat := targetFloat != nil
	hasValues := targetValues != nil
	if hasFloat == hasValues {
		diagnostics.AddAttributeError(path.Root("target_condition"), "Invalid target condition", "Set exactly one of target_float or target_values.")
		return nil, diagnostics
	}
	operator := comparisonOperator.ValueString()
	isMembership := operator == "in" || operator == "nin"
	if isMembership != hasValues {
		diagnostics.AddAttributeError(path.Root("target_condition"), "Invalid target condition", "Use target_values with in or nin, and target_float with eq, neq, gt, gte, lt, or lte.")
		return nil, diagnostics
	}
	return &client.MetricTargetCondition{ComparisonOperator: operator, TargetFloat: targetFloat, TargetValues: targetValues}, diagnostics
}

func metricResourceState(ctx context.Context, remote client.Metric, prior *metricResourceModel) (metricResourceModel, diag.Diagnostics) {
	return metricState(ctx, remote, prior)
}
func metricDataSourceState(ctx context.Context, remote client.Metric) (metricResourceModel, diag.Diagnostics) {
	return metricState(ctx, remote, nil)
}

func metricState(ctx context.Context, remote client.Metric, prior *metricResourceModel) (metricResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	set := func(values *[]string) types.Set {
		if values == nil {
			return types.SetNull(types.StringType)
		}
		result, d := types.SetValueFrom(ctx, types.StringType, *values)
		diagnostics.Append(d...)
		return result
	}
	tags, d := types.SetValueFrom(ctx, types.StringType, remote.Tags)
	diagnostics.Append(d...)
	expectedBody, err := dynamicFromJSONValuePreserving(remote.ExpectedBody, priorMetricDynamic(prior, func(model *metricResourceModel) types.Dynamic { return model.ExpectedBody }))
	if err != nil {
		diagnostics.AddError("Unable to decode metric expected body", err.Error())
	}
	ivrFlow := types.DynamicNull()
	if len(remote.IVRFlow) > 0 && !bytes.Equal(bytes.TrimSpace(remote.IVRFlow), []byte("null")) {
		ivrFlow, err = dynamicFromJSONObjectPreserving(remote.IVRFlow, priorMetricDynamic(prior, func(model *metricResourceModel) types.Dynamic { return model.IVRFlow }))
		if err != nil {
			diagnostics.AddError("Unable to decode metric IVR flow", err.Error())
		}
	}
	state := metricResourceModel{
		Name: types.StringValue(remote.Name), ID: types.StringValue(remote.ID), MetricName: types.StringValue(remote.MetricName),
		Description: types.StringValue(remote.Description), MetricType: types.StringValue(remote.MetricType), Evaluation: metricEvaluationValue(remote.Evaluation),
		Prompt: nullableMetricString(remote.Prompt), EnabledTools: set(remote.EnabledTools), Categories: set(remote.Categories),
		MinValue: nullableFloat(remote.MinValue), MaxValue: nullableFloat(remote.MaxValue), MetadataFieldType: nullableMetricString(remote.MetadataFieldType),
		MetadataFieldKey: nullableMetricString(remote.MetadataFieldKey), RegexPattern: nullableMetricString(remote.RegexPattern), Role: metricRoleState(remote.Role, prior),
		MinPauseDurationSeconds: nullableFloat(remote.MinPauseDurationSeconds), MaxSilenceDurationSeconds: nullableFloat(remote.MaxSilenceDurationSeconds),
		MinSilenceGapSeconds: nullableFloat(remote.MinSilenceGapSeconds), FrequencyThreshold: nullableFloat(remote.FrequencyThreshold),
		Direction: nullableMetricString(remote.Direction), SuccessSentiments: set(remote.SuccessSentiments), PercentAbove: nullableFloat(remote.PercentAbove),
		SuccessEndReasons: set(remote.SuccessEndReasons), ObservationName: nullableMetricString(remote.ObservationName), ExpectedBody: expectedBody,
		MatchPath: nullableMetricString(remote.MatchPath), MinVolumeChangeForPitchMisalignment: nullableFloat(remote.MinVolumeChangeForPitchMisalignment),
		Threshold: nullableInt(remote.Threshold), Operator: nullableMetricString(remote.Operator), IVRFlow: ivrFlow, SQLQuery: nullableMetricString(remote.SQLQuery),
		CriteriaSource: nullableMetricString(remote.CriteriaSource), CriteriaPath: nullableMetricString(remote.CriteriaPath), Criteria: set(remote.Criteria),
		ReportingMethod: nullableMetricString(remote.ReportingMethod), BasePromptTemplate: nullableMetricString(remote.BasePromptTemplate),
		IncludeTraces: nullableBool(remote.IncludeTraces), RuntimeConfig: runtimeConfigValue(remote.RuntimeConfig), TargetCondition: targetConditionValue(ctx, remote.TargetCondition, &diagnostics),
		Tags: tags, CreatedBy: nullableMetricString(remote.CreatedBy), CreateTime: types.StringValue(remote.CreateTime), UpdateTime: nullableMetricString(remote.UpdateTime),
		CurrentVersion: currentMetricVersionValue(remote.CurrentVersion),
	}
	return state, diagnostics
}

func priorMetricDynamic(prior *metricResourceModel, selectValue func(*metricResourceModel) types.Dynamic) types.Dynamic {
	if prior == nil {
		return types.DynamicNull()
	}
	return selectValue(prior)
}
func nullableMetricString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func metricRoleState(value *string, prior *metricResourceModel) types.String {
	if value == nil {
		return types.StringNull()
	}
	if prior != nil && !prior.Role.IsNull() && !prior.Role.IsUnknown() {
		configured := prior.Role.ValueString()
		canonical := configured
		switch configured {
		case "assistant":
			canonical = "agent"
		case "user":
			canonical = "persona"
		}
		if canonical == *value {
			return prior.Role
		}
	}
	return types.StringValue(*value)
}

func nullableFloat(value *float64) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*value)
}
func nullableInt(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}
func nullableBool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func runtimeConfigValue(value *client.MetricRuntimeConfig) types.Object {
	if value == nil {
		return types.ObjectNull(metricRuntimeConfigAttributeTypes)
	}
	result, _ := types.ObjectValue(metricRuntimeConfigAttributeTypes, map[string]attr.Value{"model_version": nullableMetricString(value.ModelVersion), "thinking_enabled": nullableBool(value.ThinkingEnabled)})
	return result
}

func targetConditionValue(ctx context.Context, value *client.MetricTargetCondition, diagnostics *diag.Diagnostics) types.Object {
	if value == nil {
		return types.ObjectNull(metricTargetConditionAttributeTypes)
	}
	targetValues := types.SetNull(types.StringType)
	if value.TargetValues != nil {
		var d diag.Diagnostics
		targetValues, d = types.SetValueFrom(ctx, types.StringType, *value.TargetValues)
		diagnostics.Append(d...)
	}
	result, d := types.ObjectValue(metricTargetConditionAttributeTypes, map[string]attr.Value{"comparison_operator": types.StringValue(value.ComparisonOperator), "target_float": nullableFloat(value.TargetFloat), "target_values": targetValues})
	diagnostics.Append(d...)
	return result
}

func metricEvaluationValue(value *client.MetricEvaluation) types.Object {
	if value == nil {
		return types.ObjectNull(metricEvaluationAttributeTypes)
	}
	result, _ := types.ObjectValue(metricEvaluationAttributeTypes, map[string]attr.Value{"evaluator": types.StringValue(value.Evaluator), "output_type": nullableMetricString(value.OutputType), "output_type_source": types.StringValue(value.OutputTypeSource), "semantic_type": nullableMetricString(value.SemanticType), "semantic_type_source": types.StringValue(value.SemanticTypeSource)})
	return result
}

func currentMetricVersionValue(value *client.CurrentMetricVersion) types.Object {
	if value == nil {
		return types.ObjectNull(currentMetricVersionAttributeTypes)
	}
	result, _ := types.ObjectValue(currentMetricVersionAttributeTypes, map[string]attr.Value{"ulid": types.StringValue(value.ULID), "version_number": types.Int64Value(value.VersionNumber), "change_type": types.StringValue(value.ChangeType)})
	return result
}
