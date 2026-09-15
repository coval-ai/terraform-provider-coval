package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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
