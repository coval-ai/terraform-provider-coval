package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceSchema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMetricResourceSchemaAndInput(t *testing.T) {
	t.Parallel()
	var response resource.SchemaResponse
	(&metricResource{}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	for _, name := range []string{"metric_name", "description", "metric_type"} {
		attribute, ok := response.Schema.Attributes[name].(schema.StringAttribute)
		if !ok || !attribute.Required {
			t.Errorf("%s must be a required string", name)
		}
	}
	role, ok := response.Schema.Attributes["role"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("role has type %T, want schema.StringAttribute", response.Schema.Attributes["role"])
	}
	for _, alias := range []string{"user", "assistant"} {
		var validation validator.StringResponse
		role.Validators[0].ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(alias)}, &validation)
		if validation.Diagnostics.HasError() {
			t.Errorf("role validator rejected public alias %q: %v", alias, validation.Diagnostics)
		}
	}
	tags, ok := response.Schema.Attributes["tags"].(schema.SetAttribute)
	if !ok {
		t.Fatalf("tags has type %T, want schema.SetAttribute", response.Schema.Attributes["tags"])
	}
	if len(tags.Validators) != 0 {
		t.Fatalf("metric tags have %d validators, public write contract has no cardinality limit", len(tags.Validators))
	}
	target := types.ObjectValueMust(metricTargetConditionAttributeTypes, map[string]attr.Value{"comparison_operator": types.StringValue("in"), "target_float": types.Float64Null(), "target_values": types.SetValueMust(types.StringType, []attr.Value{types.StringValue("YES")})})
	plan := metricResourceModel{MetricName: types.StringValue("Resolution"), Description: types.StringValue("Whether the issue was resolved"), MetricType: types.StringValue("METRIC_LLM_BINARY"), TargetCondition: target, RuntimeConfig: types.ObjectNull(metricRuntimeConfigAttributeTypes), ExpectedBody: types.DynamicNull(), IVRFlow: types.DynamicNull(), EnabledTools: types.SetNull(types.StringType), Categories: types.SetNull(types.StringType), SuccessSentiments: types.SetNull(types.StringType), SuccessEndReasons: types.SetNull(types.StringType), Criteria: types.SetNull(types.StringType), Tags: types.SetNull(types.StringType)}
	input, diagnostics := metricInput(context.Background(), plan)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if input.TargetCondition == nil || input.TargetCondition.ComparisonOperator != "in" || input.TargetCondition.TargetValues == nil {
		t.Fatalf("target condition = %#v", input.TargetCondition)
	}
}

func TestMetricTargetConditionShapes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		operator     string
		targetFloat  types.Float64
		targetValues types.Set
		wantError    bool
	}{
		{name: "membership", operator: "in", targetFloat: types.Float64Null(), targetValues: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("YES")})},
		{name: "numeric", operator: "gte", targetFloat: types.Float64Value(4), targetValues: types.SetNull(types.StringType)},
		{name: "neither target", operator: "eq", targetFloat: types.Float64Null(), targetValues: types.SetNull(types.StringType), wantError: true},
		{name: "both targets", operator: "eq", targetFloat: types.Float64Value(1), targetValues: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("YES")}), wantError: true},
		{name: "membership with float", operator: "in", targetFloat: types.Float64Value(1), targetValues: types.SetNull(types.StringType), wantError: true},
		{name: "numeric with values", operator: "gt", targetFloat: types.Float64Null(), targetValues: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("1")}), wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := types.ObjectValueMust(metricTargetConditionAttributeTypes, map[string]attr.Value{
				"comparison_operator": types.StringValue(test.operator),
				"target_float":        test.targetFloat,
				"target_values":       test.targetValues,
			})
			_, diagnostics := targetConditionFromObject(t.Context(), value)
			if diagnostics.HasError() != test.wantError {
				t.Fatalf("diagnostics = %v, want error = %t", diagnostics, test.wantError)
			}
		})
	}
}

func TestMetricRuntimeConfigAndListTagLimits(t *testing.T) {
	t.Parallel()
	empty := types.ObjectValueMust(metricRuntimeConfigAttributeTypes, map[string]attr.Value{
		"model_version":    types.StringNull(),
		"thinking_enabled": types.BoolNull(),
	})
	if !runtimeConfigIsExplicitlyEmpty(empty) {
		t.Fatal("empty configured runtime_config was not recognized as an explicit reset")
	}
	configured := types.ObjectValueMust(metricRuntimeConfigAttributeTypes, map[string]attr.Value{
		"model_version":    types.StringValue("openai:gpt-4.1-mini-2025-04-14"),
		"thinking_enabled": types.BoolNull(),
	})
	if runtimeConfigIsExplicitlyEmpty(configured) || runtimeConfigIsExplicitlyEmpty(types.ObjectNull(metricRuntimeConfigAttributeTypes)) {
		t.Fatal("configured or omitted runtime_config was recognized as an explicit reset")
	}

	var response datasource.SchemaResponse
	(&metricsDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &response)
	tagFilters, ok := response.Schema.Attributes["tag_filters"].(datasourceSchema.SetAttribute)
	if !ok {
		t.Fatalf("tag_filters has type %T, want schema.SetAttribute", response.Schema.Attributes["tag_filters"])
	}
	if len(tagFilters.Validators) != 1 {
		t.Fatalf("tag_filters has %d validators, want public API max-items validator", len(tagFilters.Validators))
	}
}

func TestMetricStatePreservesPolymorphicFields(t *testing.T) {
	t.Parallel()
	expected := json.RawMessage(`{"status":"ok"}`)
	canonicalRole := "agent"
	prior := &metricResourceModel{Role: types.StringValue("assistant")}
	state, diagnostics := metricResourceState(context.Background(), client.Metric{Name: "metrics/abc123def456ghi789jklm", ID: "abc123def456ghi789jklm", MetricName: "Body", Description: "Matches output", MetricType: "METRIC_MATCH_EXPECTED_OUTPUT", Role: &canonicalRole, ExpectedBody: expected, IVRFlow: json.RawMessage(`null`), Tags: []string{}, CreateTime: "2026-09-18T00:00:00Z"}, prior)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if state.ExpectedBody.IsNull() || !state.IVRFlow.IsNull() || state.CurrentVersion.IsNull() == false || state.Role.ValueString() != "assistant" {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestMetricExpectedBodyRejectsUnsupportedShapes(t *testing.T) {
	t.Parallel()
	if _, err := metricExpectedBody(types.DynamicValue(types.BoolValue(true))); err == nil {
		t.Fatal("metricExpectedBody accepted a boolean")
	}
	if _, err := metricExpectedBody(types.DynamicValue(types.StringValue(""))); err == nil {
		t.Fatal("metricExpectedBody accepted an empty string")
	}
}

func TestListAllMetricsRejectsRepeatedToken(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"metrics":[],"next_page_token":"same-token"}`)
	}))
	defer server.Close()
	apiClient, err := client.New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := listAllMetrics(t.Context(), apiClient, client.ListMetricsOptions{PageSize: 100}); err == nil {
		t.Fatal("listAllMetrics accepted a repeated token")
	}
}
