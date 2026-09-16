package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDynamicJSONObjectRoundTrip(t *testing.T) {
	t.Parallel()

	nested, diagnostics := types.ObjectValue(
		map[string]attr.Type{"enabled": types.BoolType},
		map[string]attr.Value{"enabled": types.BoolValue(true)},
	)
	if diagnostics.HasError() {
		t.Fatalf("create nested object: %v", diagnostics)
	}
	value, diagnostics := types.ObjectValue(
		map[string]attr.Type{"name": types.StringType, "nested": nested.Type(t.Context())},
		map[string]attr.Value{"name": types.StringValue("terraform"), "nested": nested},
	)
	if diagnostics.HasError() {
		t.Fatalf("create object: %v", diagnostics)
	}

	raw, err := dynamicJSONObject(types.DynamicValue(value))
	if err != nil {
		t.Fatalf("dynamicJSONObject(): %v", err)
	}
	if raw == nil || !jsonValuesEqual(*raw, json.RawMessage(`{"name":"terraform","nested":{"enabled":true}}`)) {
		t.Fatalf("encoded object = %s", raw)
	}

	roundTrip, err := dynamicFromJSONObject(*raw)
	if err != nil {
		t.Fatalf("dynamicFromJSONObject(): %v", err)
	}
	if !roundTrip.UnderlyingValue().Equal(value) {
		t.Errorf("round trip = %s, want %s", roundTrip, value)
	}
}

func TestDynamicJSONObjectRejectsNonObject(t *testing.T) {
	t.Parallel()
	_, err := dynamicJSONObject(types.DynamicValue(types.StringValue("not-an-object")))
	if err == nil {
		t.Fatal("dynamicJSONObject() accepted a string")
	}
}

func TestDynamicJSONObjectRejectsUnknownNestedValue(t *testing.T) {
	t.Parallel()
	value, diagnostics := types.ObjectValue(
		map[string]attr.Type{"value": types.StringType},
		map[string]attr.Value{"value": types.StringUnknown()},
	)
	if diagnostics.HasError() {
		t.Fatalf("create object: %v", diagnostics)
	}
	_, err := dynamicJSONObject(types.DynamicValue(value))
	if err == nil {
		t.Fatal("dynamicJSONObject() accepted an unknown nested value")
	}
}

func TestDynamicJSONObjectHasKeyDoesNotRequireKnownValues(t *testing.T) {
	t.Parallel()

	value, diagnostics := types.ObjectValue(
		map[string]attr.Type{
			"generated_url": types.StringType,
			"script_turns":  types.StringType,
		},
		map[string]attr.Value{
			"generated_url": types.StringUnknown(),
			"script_turns":  types.StringUnknown(),
		},
	)
	if diagnostics.HasError() {
		t.Fatalf("create object: %v", diagnostics)
	}

	hasKey, err := dynamicJSONObjectHasKey(types.DynamicValue(value), "script_turns")
	if err != nil {
		t.Fatalf("dynamicJSONObjectHasKey(): %v", err)
	}
	if !hasKey {
		t.Error("dynamicJSONObjectHasKey() = false, want true")
	}
}

func TestDynamicJSONObjectPreservesEquivalentTerraformType(t *testing.T) {
	t.Parallel()
	value, diagnostics := types.MapValue(
		types.StringType,
		map[string]attr.Value{"owner": types.StringValue("terraform")},
	)
	if diagnostics.HasError() {
		t.Fatalf("create map: %v", diagnostics)
	}
	prior := types.DynamicValue(value)
	result, err := dynamicFromJSONObjectPreserving(json.RawMessage(`{"owner":"terraform"}`), prior)
	if err != nil {
		t.Fatalf("dynamicFromJSONObjectPreserving(): %v", err)
	}
	if !result.Equal(prior) {
		t.Errorf("result = %s, want preserved %s", result, prior)
	}
}

func TestJSONObjectWithoutKey(t *testing.T) {
	t.Parallel()

	result, err := jsonObjectWithoutKey(
		json.RawMessage(`{"channel":"voice","script_turns":[{"type":"text","text":"Hello"}]}`),
		"script_turns",
	)
	if err != nil {
		t.Fatalf("jsonObjectWithoutKey(): %v", err)
	}
	if want := json.RawMessage(`{"channel":"voice"}`); !jsonValuesEqual(result, want) {
		t.Errorf("result = %s, want %s", result, want)
	}
}

func TestDynamicJSONArrayRoundTrip(t *testing.T) {
	t.Parallel()

	skip, diagnostics := types.ObjectValue(
		map[string]attr.Type{"type": types.StringType},
		map[string]attr.Value{"type": types.StringValue("skip")},
	)
	if diagnostics.HasError() {
		t.Fatalf("create skip turn: %v", diagnostics)
	}
	value, diagnostics := types.TupleValue(
		[]attr.Type{types.StringType, skip.Type(t.Context())},
		[]attr.Value{types.StringValue("Hello"), skip},
	)
	if diagnostics.HasError() {
		t.Fatalf("create script turns: %v", diagnostics)
	}

	raw, err := dynamicJSONArray(types.DynamicValue(value))
	if err != nil {
		t.Fatalf("dynamicJSONArray(): %v", err)
	}
	if raw == nil || !jsonValuesEqual(*raw, json.RawMessage(`["Hello",{"type":"skip"}]`)) {
		t.Fatalf("encoded script turns = %s", raw)
	}

	roundTrip, err := dynamicFromJSONArrayPreserving(*raw, types.DynamicValue(value))
	if err != nil {
		t.Fatalf("dynamicFromJSONArrayPreserving(): %v", err)
	}
	if !roundTrip.UnderlyingValue().Equal(value) {
		t.Errorf("round trip = %s, want %s", roundTrip, value)
	}
}

func TestDynamicJSONArrayRejectsNonArray(t *testing.T) {
	t.Parallel()
	_, err := dynamicJSONArray(types.DynamicValue(types.StringValue("not-an-array")))
	if err == nil {
		t.Fatal("dynamicJSONArray() accepted a string")
	}
}

func TestDynamicJSONArrayLengthDoesNotRequireKnownElements(t *testing.T) {
	t.Parallel()

	value, diagnostics := types.TupleValue(
		[]attr.Type{types.StringType},
		[]attr.Value{types.StringUnknown()},
	)
	if diagnostics.HasError() {
		t.Fatalf("create tuple: %v", diagnostics)
	}

	length, err := dynamicJSONArrayLength(types.DynamicValue(value))
	if err != nil {
		t.Fatalf("dynamicJSONArrayLength(): %v", err)
	}
	if length != 1 {
		t.Errorf("dynamicJSONArrayLength() = %d, want 1", length)
	}
}

func TestDynamicFromJSONArrayPreservesNull(t *testing.T) {
	t.Parallel()
	value, err := dynamicFromJSONArray(json.RawMessage(`null`))
	if err != nil {
		t.Fatalf("dynamicFromJSONArray(): %v", err)
	}
	if !value.IsNull() {
		t.Errorf("value = %s, want null", value)
	}
}

func TestDynamicFromJSONArrayPreservesConfiguredEmptyTupleWhenResponseIsNull(t *testing.T) {
	t.Parallel()

	emptyTuple, diagnostics := types.TupleValue([]attr.Type{}, []attr.Value{})
	if diagnostics.HasError() {
		t.Fatalf("create empty tuple: %v", diagnostics)
	}
	prior := types.DynamicValue(emptyTuple)

	value, err := dynamicFromJSONArrayPreserving(json.RawMessage(`null`), prior)
	if err != nil {
		t.Fatalf("dynamicFromJSONArrayPreserving(): %v", err)
	}
	if !value.Equal(prior) {
		t.Errorf("value = %s, want preserved %s", value, prior)
	}
}
