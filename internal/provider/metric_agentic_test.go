package provider

import (
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAgenticJudgeConfiguration(t *testing.T) {
	t.Parallel()
	pairs := []struct{ text, agentic string }{
		{"METRIC_LLM_BINARY", "METRIC_AGENT_JUDGE"},
		{"METRIC_CATEGORICAL", "METRIC_AGENT_JUDGE_CATEGORICAL"},
		{"METRIC_NUMERICAL_LLM_JUDGE", "METRIC_AGENT_JUDGE_NUMERICAL"},
	}
	var response resource.SchemaResponse
	(&metricResource{}).Schema(t.Context(), resource.SchemaRequest{}, &response)
	metricType, ok := response.Schema.Attributes["metric_type"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("metric_type has unexpected schema type %T", response.Schema.Attributes["metric_type"])
	}
	for _, pair := range pairs {
		for _, configuredType := range []string{pair.text, pair.agentic} {
			t.Run(configuredType, func(t *testing.T) {
				var validation validator.StringResponse
				metricType.Validators[0].ValidateString(t.Context(), validator.StringRequest{ConfigValue: types.StringValue(configuredType)}, &validation)
				if validation.Diagnostics.HasError() {
					t.Fatal(validation.Diagnostics)
				}
				plan := metricResourceModel{
					MetricType: types.StringValue(configuredType), JudgeMode: types.StringValue("AGENTIC"),
					Prompt:       types.StringValue("Evaluate resolution using evidence."),
					EnabledTools: types.SetValueMust(types.StringType, []attr.Value{}),
					Categories:   types.SetNull(types.StringType), SuccessSentiments: types.SetNull(types.StringType),
					SuccessEndReasons: types.SetNull(types.StringType), Criteria: types.SetNull(types.StringType), Tags: types.SetNull(types.StringType),
					RuntimeConfig: types.ObjectNull(metricRuntimeConfigAttributeTypes), TargetCondition: types.ObjectNull(metricTargetConditionAttributeTypes),
					ExpectedBody: types.DynamicNull(), IVRFlow: types.DynamicNull(),
				}
				input, diagnostics := metricInput(t.Context(), plan)
				if diagnostics.HasError() {
					t.Fatal(diagnostics)
				}
				if input.JudgeMode == nil || *input.JudgeMode != "AGENTIC" || input.EnabledTools == nil || len(*input.EnabledTools) != 0 {
					t.Fatalf("agentic input = %#v", input)
				}
				mode := "AGENTIC"
				remote := client.Metric{MetricType: pair.agentic, JudgeMode: &mode, EnabledTools: input.EnabledTools, Tags: []string{}}
				state, diagnostics := metricResourceState(t.Context(), remote, &plan)
				if diagnostics.HasError() {
					t.Fatal(diagnostics)
				}
				if state.MetricType.ValueString() != configuredType || state.JudgeMode.ValueString() != mode || state.EnabledTools.IsNull() || len(state.EnabledTools.Elements()) != 0 {
					t.Fatalf("agentic state = %#v", state)
				}
				imported, diagnostics := metricState(t.Context(), remote, nil)
				if diagnostics.HasError() || imported.MetricType.ValueString() != pair.agentic || imported.JudgeMode.ValueString() != mode {
					t.Fatalf("imported state = %#v, diagnostics = %v", imported, diagnostics)
				}
				remote.MetricType, mode = pair.text, "STANDARD"
				state, diagnostics = metricResourceState(t.Context(), remote, &plan)
				if diagnostics.HasError() || state.MetricType.ValueString() != pair.text || state.JudgeMode.ValueString() != "STANDARD" {
					t.Fatalf("standard state = %#v, diagnostics = %v", state, diagnostics)
				}
			})
		}
	}
	mode, ok := response.Schema.Attributes["judge_mode"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("judge_mode has unexpected schema type %T", response.Schema.Attributes["judge_mode"])
	}
	for _, value := range []string{"STANDARD", "AGENTIC", "invalid"} {
		var validation validator.StringResponse
		mode.Validators[0].ValidateString(t.Context(), validator.StringRequest{ConfigValue: types.StringValue(value)}, &validation)
		if validation.Diagnostics.HasError() != (value == "invalid") {
			t.Fatalf("mode %q: %v", value, validation.Diagnostics)
		}
	}
}
