package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTestCaseResourceMutableTestSetID(t *testing.T) {
	t.Parallel()

	var response resource.SchemaResponse
	newTestCaseResource().Schema(context.Background(), resource.SchemaRequest{}, &response)

	attribute, ok := response.Schema.Attributes["test_set_id"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("test_set_id has type %T, want schema.StringAttribute", response.Schema.Attributes["test_set_id"])
	}
	if !attribute.Required {
		t.Error("test_set_id must be required")
	}
	if len(attribute.PlanModifiers) != 0 {
		t.Errorf("test_set_id has %d plan modifiers, want none so it remains mutable", len(attribute.PlanModifiers))
	}
}

func TestTestCaseResourceCollectionAttributesUseStateForUnknown(t *testing.T) {
	t.Parallel()

	var response resource.SchemaResponse
	newTestCaseResource().Schema(context.Background(), resource.SchemaRequest{}, &response)

	behaviors, ok := response.Schema.Attributes["expected_behaviors"].(schema.ListAttribute)
	if !ok {
		t.Fatalf("expected_behaviors has type %T, want schema.ListAttribute", response.Schema.Attributes["expected_behaviors"])
	}
	if len(behaviors.PlanModifiers) != 1 {
		t.Fatalf("expected_behaviors has %d plan modifiers, want 1", len(behaviors.PlanModifiers))
	}
	if got, want := reflect.TypeOf(behaviors.PlanModifiers[0]), reflect.TypeOf(listplanmodifier.UseStateForUnknown()); got != want {
		t.Errorf("expected_behaviors plan modifier has type %v, want %v", got, want)
	}

	wantDynamicModifier := reflect.TypeOf(dynamicplanmodifier.UseStateForUnknown())
	for _, name := range []string{"expected_output_json", "script_turns", "simulation_metadata_input", "metric_input"} {
		attribute, ok := response.Schema.Attributes[name].(schema.DynamicAttribute)
		if !ok {
			t.Fatalf("%s has type %T, want schema.DynamicAttribute", name, response.Schema.Attributes[name])
		}
		if len(attribute.PlanModifiers) != 1 {
			t.Fatalf("%s has %d plan modifiers, want 1", name, len(attribute.PlanModifiers))
		}
		if got := reflect.TypeOf(attribute.PlanModifiers[0]); got != wantDynamicModifier {
			t.Errorf("%s plan modifier has type %v, want %v", name, got, wantDynamicModifier)
		}
	}
}

func TestTestCaseResourceInputTypeMatchesPublicContract(t *testing.T) {
	t.Parallel()

	var response resource.SchemaResponse
	newTestCaseResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	attribute, ok := response.Schema.Attributes["input_type"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("input_type has type %T, want schema.StringAttribute", response.Schema.Attributes["input_type"])
	}

	for _, value := range []string{"CUSTOM", strings.Repeat("x", 200)} {
		if diagnostics := validateString(t.Context(), attribute.Validators, value); diagnostics.HasError() {
			t.Errorf("input_type %q was rejected: %v", value, diagnostics)
		}
	}
	if diagnostics := validateString(t.Context(), attribute.Validators, strings.Repeat("x", 201)); !diagnostics.HasError() {
		t.Error("input_type longer than 200 characters was accepted")
	}
}

func TestValidateTestCaseScriptConfig(t *testing.T) {
	t.Parallel()

	nonEmptyTuple, diagnostics := types.TupleValue(
		[]attr.Type{types.StringType},
		[]attr.Value{types.StringValue("Hello")},
	)
	if diagnostics.HasError() {
		t.Fatalf("create non-empty script turns: %v", diagnostics)
	}
	emptyTuple, diagnostics := types.TupleValue([]attr.Type{}, []attr.Value{})
	if diagnostics.HasError() {
		t.Fatalf("create empty script turns: %v", diagnostics)
	}
	legacySimulationMetadata, diagnostics := types.ObjectValue(
		map[string]attr.Type{"script_turns": nonEmptyTuple.Type(t.Context())},
		map[string]attr.Value{"script_turns": nonEmptyTuple},
	)
	if diagnostics.HasError() {
		t.Fatalf("create legacy simulation metadata: %v", diagnostics)
	}
	metadataWithUnknownValue, diagnostics := types.ObjectValue(
		map[string]attr.Type{"generated_url": types.StringType},
		map[string]attr.Value{"generated_url": types.StringUnknown()},
	)
	if diagnostics.HasError() {
		t.Fatalf("create simulation metadata with unknown value: %v", diagnostics)
	}
	turnsWithUnknownValue, diagnostics := types.TupleValue(
		[]attr.Type{types.StringType},
		[]attr.Value{types.StringUnknown()},
	)
	if diagnostics.HasError() {
		t.Fatalf("create script turns with unknown value: %v", diagnostics)
	}

	tests := map[string]struct {
		config    testCaseResourceModel
		wantError bool
	}{
		"SCRIPT with turns": {
			config: testCaseResourceModel{
				InputType:   types.StringValue("SCRIPT"),
				ScriptTurns: types.DynamicValue(nonEmptyTuple),
			},
		},
		"SCRIPT without turns": {
			config: testCaseResourceModel{
				InputType:   types.StringValue("SCRIPT"),
				ScriptTurns: types.DynamicNull(),
			},
			wantError: true,
		},
		"SCRIPT with empty turns": {
			config: testCaseResourceModel{
				InputType:   types.StringValue("SCRIPT"),
				ScriptTurns: types.DynamicValue(emptyTuple),
			},
			wantError: true,
		},
		"non-SCRIPT without turns": {
			config: testCaseResourceModel{
				InputType:   types.StringValue("SCENARIO"),
				ScriptTurns: types.DynamicNull(),
			},
		},
		"non-SCRIPT with turns": {
			config: testCaseResourceModel{
				InputType:   types.StringValue("SCENARIO"),
				ScriptTurns: types.DynamicValue(nonEmptyTuple),
			},
			wantError: true,
		},
		"legacy nested turns": {
			config: testCaseResourceModel{
				InputType:          types.StringValue("SCRIPT"),
				ScriptTurns:        types.DynamicValue(nonEmptyTuple),
				SimulationMetadata: types.DynamicValue(legacySimulationMetadata),
			},
			wantError: true,
		},
		"metadata with resource-derived value": {
			config: testCaseResourceModel{
				InputType:          types.StringValue("SCENARIO"),
				ScriptTurns:        types.DynamicNull(),
				SimulationMetadata: types.DynamicValue(metadataWithUnknownValue),
			},
		},
		"script turns with resource-derived value": {
			config: testCaseResourceModel{
				InputType:   types.StringValue("SCRIPT"),
				ScriptTurns: types.DynamicValue(turnsWithUnknownValue),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			diagnostics := validateTestCaseScriptConfig(test.config)
			if got := diagnostics.HasError(); got != test.wantError {
				t.Errorf("HasError() = %t, want %t: %v", got, test.wantError, diagnostics)
			}
		})
	}
}

func TestTestCasePlanClearsUnconfiguredTurnsForNonScriptInput(t *testing.T) {
	t.Parallel()

	turnsTuple, diagnostics := types.TupleValue(
		[]attr.Type{types.StringType},
		[]attr.Value{types.StringValue("Hello")},
	)
	if diagnostics.HasError() {
		t.Fatalf("create script turns: %v", diagnostics)
	}
	turns := types.DynamicValue(turnsTuple)
	config := testCaseResourceModel{ScriptTurns: types.DynamicNull()}

	nonScriptPlan := testCasePlanForConfig(config, testCaseResourceModel{
		InputType:   types.StringValue("SCENARIO"),
		ScriptTurns: turns,
	})
	if !nonScriptPlan.ScriptTurns.IsNull() {
		t.Errorf("non-SCRIPT script_turns = %s, want null", nonScriptPlan.ScriptTurns)
	}

	scriptPlan := testCasePlanForConfig(config, testCaseResourceModel{
		InputType:   types.StringValue("SCRIPT"),
		ScriptTurns: turns,
	})
	if !scriptPlan.ScriptTurns.Equal(turns) {
		t.Errorf("SCRIPT script_turns = %s, want %s", scriptPlan.ScriptTurns, turns)
	}
}

func validateString(ctx context.Context, validators []validator.String, value string) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	for _, item := range validators {
		var response validator.StringResponse
		item.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue(value)}, &response)
		diagnostics.Append(response.Diagnostics...)
	}
	return diagnostics
}

func TestTestCaseResourceStateMapsPublicAPIResponse(t *testing.T) {
	t.Parallel()

	testSetID := "abc12345"
	description := "Refund policy"
	inputType := "SCRIPT"
	userNotes := "Regression coverage"
	updateTime := "2026-09-15T01:00:00Z"
	expectedBehaviors := []string{"Explain the policy", "Offer next steps"}
	remote := client.TestCase{
		Name:               "test-cases/abc123def456ghi789jklm",
		ID:                 "abc123def456ghi789jklm",
		TestSetID:          &testSetID,
		InputString:        "Ask about the refund policy",
		ExpectedBehaviors:  &expectedBehaviors,
		ExpectedOutputJSON: json.RawMessage(`{"eligible":true}`),
		Description:        &description,
		InputType:          &inputType,
		ScriptTurns:        json.RawMessage(`[{"type":"text","text":"Hello"}]`),
		SimulationMetadata: json.RawMessage(`{"channel":"voice","script_turns":[{"type":"text","text":"Hello"}]}`),
		MetricInput:        json.RawMessage(`{"policy_window_days":30}`),
		UserNotes:          &userNotes,
		CreateTime:         "2026-09-15T00:00:00Z",
		UpdateTime:         &updateTime,
	}

	state, diagnostics := testCaseResourceState(t.Context(), remote, nil)
	if diagnostics.HasError() {
		t.Fatalf("testCaseResourceState(): %v", diagnostics)
	}

	if got, want := state.ID, types.StringValue(remote.ID); !got.Equal(want) {
		t.Errorf("id = %s, want %s", got, want)
	}
	if got, want := state.Name, types.StringValue(remote.Name); !got.Equal(want) {
		t.Errorf("name = %s, want %s", got, want)
	}
	if got, want := state.TestSetID, types.StringValue(testSetID); !got.Equal(want) {
		t.Errorf("test_set_id = %s, want %s", got, want)
	}
	if got, want := state.InputString, types.StringValue(remote.InputString); !got.Equal(want) {
		t.Errorf("input_str = %s, want %s", got, want)
	}
	if got, want := state.Description, types.StringValue(description); !got.Equal(want) {
		t.Errorf("description = %s, want %s", got, want)
	}
	if got, want := state.InputType, types.StringValue(inputType); !got.Equal(want) {
		t.Errorf("input_type = %s, want %s", got, want)
	}
	if got, want := state.UserNotes, types.StringValue(userNotes); !got.Equal(want) {
		t.Errorf("user_notes = %s, want %s", got, want)
	}
	if got, want := state.CreateTime, types.StringValue(remote.CreateTime); !got.Equal(want) {
		t.Errorf("create_time = %s, want %s", got, want)
	}
	if got, want := state.UpdateTime, types.StringValue(updateTime); !got.Equal(want) {
		t.Errorf("update_time = %s, want %s", got, want)
	}

	behaviors, behaviorDiagnostics := types.ListValueFrom(t.Context(), types.StringType, expectedBehaviors)
	if behaviorDiagnostics.HasError() {
		t.Fatalf("create expected behaviors: %v", behaviorDiagnostics)
	}
	if !state.ExpectedBehaviors.Equal(behaviors) {
		t.Errorf("expected_behaviors = %s, want %s", state.ExpectedBehaviors, behaviors)
	}
	assertDynamicJSONObject(t, "expected_output_json", state.ExpectedOutputJSON, remote.ExpectedOutputJSON)
	assertDynamicJSONArray(t, "script_turns", state.ScriptTurns, remote.ScriptTurns)
	assertDynamicJSONObject(t, "simulation_metadata_input", state.SimulationMetadata, json.RawMessage(`{"channel":"voice"}`))
	assertDynamicJSONObject(t, "metric_input", state.MetricInput, remote.MetricInput)
}

func TestTestCaseResourceStatePreservesConfiguredEmptyScriptTurns(t *testing.T) {
	t.Parallel()

	emptyTuple, tupleDiagnostics := types.TupleValue([]attr.Type{}, []attr.Value{})
	if tupleDiagnostics.HasError() {
		t.Fatalf("create empty tuple: %v", tupleDiagnostics)
	}
	prior := testCaseResourceModel{ScriptTurns: types.DynamicValue(emptyTuple)}
	remote := client.TestCase{
		Name:               "test-cases/abc123def456ghi789jklm",
		ID:                 "abc123def456ghi789jklm",
		InputString:        "Ask about the refund policy",
		ExpectedOutputJSON: json.RawMessage(`{}`),
		ScriptTurns:        json.RawMessage(`null`),
		SimulationMetadata: json.RawMessage(`{}`),
		MetricInput:        json.RawMessage(`{}`),
		CreateTime:         "2026-09-15T00:00:00Z",
	}

	state, diagnostics := testCaseResourceState(t.Context(), remote, &prior)
	if diagnostics.HasError() {
		t.Fatalf("testCaseResourceState(): %v", diagnostics)
	}
	if !state.ScriptTurns.Equal(prior.ScriptTurns) {
		t.Errorf("script_turns = %s, want preserved %s", state.ScriptTurns, prior.ScriptTurns)
	}
}

func TestTestCaseResourceStateMapsAbsentOptionalFieldsToNull(t *testing.T) {
	t.Parallel()

	remote := client.TestCase{
		Name:               "test-cases/abc123def456ghi789jklm",
		ID:                 "abc123def456ghi789jklm",
		InputString:        "Ask about the refund policy",
		ExpectedOutputJSON: json.RawMessage(`null`),
		ScriptTurns:        json.RawMessage(`null`),
		SimulationMetadata: json.RawMessage(`null`),
		MetricInput:        json.RawMessage(`null`),
		CreateTime:         "2026-09-15T00:00:00Z",
	}

	state, diagnostics := testCaseResourceState(t.Context(), remote, nil)
	if diagnostics.HasError() {
		t.Fatalf("testCaseResourceState(): %v", diagnostics)
	}
	for name, value := range map[string]attr.Value{
		"test_set_id":        state.TestSetID,
		"expected_behaviors": state.ExpectedBehaviors,
		"description":        state.Description,
		"input_type":         state.InputType,
		"script_turns":       state.ScriptTurns,
		"user_notes":         state.UserNotes,
		"update_time":        state.UpdateTime,
	} {
		if !value.IsNull() {
			t.Errorf("%s = %s, want null", name, value)
		}
	}
	assertDynamicJSONObject(t, "expected_output_json", state.ExpectedOutputJSON, json.RawMessage(`{}`))
	assertDynamicJSONObject(t, "simulation_metadata_input", state.SimulationMetadata, json.RawMessage(`{}`))
	assertDynamicJSONObject(t, "metric_input", state.MetricInput, json.RawMessage(`{}`))
}

func assertDynamicJSONArray(t *testing.T, name string, value types.Dynamic, want json.RawMessage) {
	t.Helper()

	got, err := dynamicJSONArray(value)
	if err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
	if got == nil || !jsonValuesEqual(*got, want) {
		t.Errorf("%s = %s, want %s", name, got, want)
	}
}
