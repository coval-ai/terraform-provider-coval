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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

func TestMetricStatePreservesPolymorphicFields(t *testing.T) {
	t.Parallel()
	expected := json.RawMessage(`{"status":"ok"}`)
	state, diagnostics := metricResourceState(context.Background(), client.Metric{Name: "metrics/abc123def456ghi789jklm", ID: "abc123def456ghi789jklm", MetricName: "Body", Description: "Matches output", MetricType: "METRIC_MATCH_EXPECTED_OUTPUT", ExpectedBody: expected, IVRFlow: json.RawMessage(`null`), Tags: []string{}, CreateTime: "2026-09-18T00:00:00Z"}, nil)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if state.ExpectedBody.IsNull() || !state.IVRFlow.IsNull() || state.CurrentVersion.IsNull() == false {
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
