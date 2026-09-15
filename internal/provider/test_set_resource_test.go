package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
)

func TestTestSetResourceDynamicAttributesUseStateForUnknown(t *testing.T) {
	t.Parallel()

	var response resource.SchemaResponse
	newTestSetResource().Schema(context.Background(), resource.SchemaRequest{}, &response)

	wantModifierType := reflect.TypeOf(dynamicplanmodifier.UseStateForUnknown())
	for _, name := range []string{"test_set_metadata", "parameters"} {
		attribute, ok := response.Schema.Attributes[name].(schema.DynamicAttribute)
		if !ok {
			t.Fatalf("%s has type %T, want schema.DynamicAttribute", name, response.Schema.Attributes[name])
		}
		if len(attribute.PlanModifiers) != 1 {
			t.Fatalf("%s has %d plan modifiers, want 1", name, len(attribute.PlanModifiers))
		}
		if got := reflect.TypeOf(attribute.PlanModifiers[0]); got != wantModifierType {
			t.Errorf("%s plan modifier has type %v, want %v", name, got, wantModifierType)
		}
	}
}
