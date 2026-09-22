package provider

import (
	"encoding/json"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAlertResourceSchemaProtectsChannelConfiguration(t *testing.T) {
	t.Parallel()
	var response resource.SchemaResponse
	newAlertResource().Schema(t.Context(), resource.SchemaRequest{}, &response)
	conditions, ok := response.Schema.Attributes["conditions"].(schema.DynamicAttribute)
	if !ok || !conditions.Required {
		t.Fatalf("conditions must be a required dynamic list: %#v", conditions)
	}
	channels, ok := response.Schema.Attributes["channels"].(schema.DynamicAttribute)
	if !ok || !channels.Optional || !channels.Computed || !channels.Sensitive {
		t.Fatalf("channels must be optional, computed, and sensitive: %#v", channels)
	}
	for _, name := range []string{"agent_ids", "required_tags", "scheduled_run_ids"} {
		attribute, ok := response.Schema.Attributes[name].(schema.SetAttribute)
		if !ok || !attribute.Optional || !attribute.Computed || attribute.Default == nil {
			t.Fatalf("%s must be optional with an empty-set default: %#v", name, attribute)
		}
	}
}

func TestAlertInputAndStateNormalizeServerIDs(t *testing.T) {
	t.Parallel()
	conditions := types.DynamicValue(types.TupleValueMust(
		[]attr.Type{types.ObjectType{AttrTypes: map[string]attr.Type{"aggregation": types.StringType, "operator": types.StringType, "threshold_float": types.Float64Type}}},
		[]attr.Value{types.ObjectValueMust(map[string]attr.Type{"aggregation": types.StringType, "operator": types.StringType, "threshold_float": types.Float64Type}, map[string]attr.Value{"aggregation": types.StringValue("RUN_AVERAGE"), "operator": types.StringValue("LT"), "threshold_float": types.Float64Value(0.9)})},
	))
	channels := types.DynamicValue(types.TupleValueMust([]attr.Type{}, []attr.Value{}))
	plan := alertResourceModel{
		WorkspaceID: types.StringValue("workspace-1"), Name: types.StringValue("Low Resolution"), Description: types.StringValue(""),
		EvaluationType: types.StringValue("ON_RUN_COMPLETE"), ConversationSource: types.StringValue("ALL"), MatchMode: types.StringValue("ALL"), CooldownSeconds: types.Int64Value(0),
		AgentIDs: types.SetValueMust(types.StringType, []attr.Value{}), RequiredTags: types.SetValueMust(types.StringType, []attr.Value{}), ScheduledRunIDs: types.SetValueMust(types.StringType, []attr.Value{}),
		Conditions: conditions, Channels: channels,
	}
	input, diagnostics := createAlertInput(t.Context(), plan)
	if diagnostics.HasError() || len(input.Conditions) == 0 || input.Channels == nil {
		t.Fatalf("input = %#v, diagnostics = %v", input, diagnostics)
	}
	state, diagnostics := alertState(t.Context(), client.Alert{
		ID: "01HZ0EXAMPLE00000000000000", Name: "Low Resolution", Status: "ACTIVE", EvaluationType: "ON_RUN_COMPLETE", ConversationSource: "ALL", MatchMode: "ALL",
		Conditions: json.RawMessage(`[{"ulid":"01HZ0CONDITION000000000000","aggregation":"RUN_AVERAGE","operator":"LT","threshold_float":0.9,"metric_id":null}]`), Channels: json.RawMessage(`[]`),
		AgentIDs: []string{}, RequiredTags: []string{}, ScheduledRunIDs: []string{}, CreateTime: "2026-09-22T00:00:00Z", UpdateTime: "2026-09-22T00:00:00Z",
	}, &plan)
	if diagnostics.HasError() {
		t.Fatalf("alertState() diagnostics = %v", diagnostics)
	}
	if state.WorkspaceID.ValueString() != "workspace-1" || !state.Conditions.Equal(conditions) {
		t.Fatalf("state = %#v", state)
	}
}
