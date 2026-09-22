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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func TestTestSetResourceValidatesPublicSlugAndTagConstraints(t *testing.T) {
	t.Parallel()

	var response resource.SchemaResponse
	newTestSetResource().Schema(context.Background(), resource.SchemaRequest{}, &response)

	slug, ok := response.Schema.Attributes["slug"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("slug has type %T, want schema.StringAttribute", response.Schema.Attributes["slug"])
	}
	if diagnostics := validateString(t.Context(), slug.Validators, "provider_tests-1"); diagnostics.HasError() {
		t.Errorf("valid slug was rejected: %v", diagnostics)
	}
	for _, value := range []string{"", "Provider-Test"} {
		if diagnostics := validateString(t.Context(), slug.Validators, value); !diagnostics.HasError() {
			t.Errorf("invalid slug %q was accepted", value)
		}
	}

	tags, ok := response.Schema.Attributes["tags"].(schema.SetAttribute)
	if !ok {
		t.Fatalf("tags has type %T, want schema.SetAttribute", response.Schema.Attributes["tags"])
	}
	validTags := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("ci")})
	if diagnostics := validateSet(t.Context(), tags.Validators, validTags); diagnostics.HasError() {
		t.Errorf("valid tags were rejected: %v", diagnostics)
	}
	invalidTags := []types.Set{
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("   ")}),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue(strings.Repeat("x", 201))}),
	}
	for _, value := range invalidTags {
		if diagnostics := validateSet(t.Context(), tags.Validators, value); !diagnostics.HasError() {
			t.Errorf("invalid tags %s were accepted", value)
		}
	}
}

func validateSet(ctx context.Context, validators []validator.Set, value types.Set) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	for _, item := range validators {
		var response validator.SetResponse
		item.ValidateSet(ctx, validator.SetRequest{ConfigValue: value}, &response)
		diagnostics.Append(response.Diagnostics...)
	}
	return diagnostics
}

func TestTestSetResourceStateMapsPublicAPIResponse(t *testing.T) {
	t.Parallel()

	description := "Managed by Terraform"
	testSetType := "SCENARIO"
	testCaseCount := int64(2)
	updateTime := "2026-09-15T01:00:00Z"
	remote := client.TestSet{
		Name:            "test-sets/abc12345",
		ID:              "abc12345",
		Slug:            "provider-test",
		DisplayName:     "Provider test",
		Description:     &description,
		TestSetType:     &testSetType,
		TestSetMetadata: json.RawMessage(`{"owner":"terraform"}`),
		Parameters:      json.RawMessage(`{"locale":["en-US"]}`),
		TestCaseCount:   &testCaseCount,
		Tags:            []string{"ci", "provider"},
		CreateTime:      "2026-09-15T00:00:00Z",
		UpdateTime:      &updateTime,
	}

	state, diagnostics := testSetResourceState(t.Context(), remote, nil)
	if diagnostics.HasError() {
		t.Fatalf("testSetResourceState(): %v", diagnostics)
	}

	if got, want := state.ID, types.StringValue(remote.ID); !got.Equal(want) {
		t.Errorf("id = %s, want %s", got, want)
	}
	if got, want := state.Name, types.StringValue(remote.Name); !got.Equal(want) {
		t.Errorf("name = %s, want %s", got, want)
	}
	if got, want := state.Slug, types.StringValue(remote.Slug); !got.Equal(want) {
		t.Errorf("slug = %s, want %s", got, want)
	}
	if got, want := state.DisplayName, types.StringValue(remote.DisplayName); !got.Equal(want) {
		t.Errorf("display_name = %s, want %s", got, want)
	}
	if got, want := state.Description, types.StringValue(description); !got.Equal(want) {
		t.Errorf("description = %s, want %s", got, want)
	}
	if got, want := state.TestSetType, types.StringValue(testSetType); !got.Equal(want) {
		t.Errorf("test_set_type = %s, want %s", got, want)
	}
	if got, want := state.TestCaseCount, types.Int64Value(testCaseCount); !got.Equal(want) {
		t.Errorf("test_case_count = %s, want %s", got, want)
	}
	if got, want := state.CreateTime, types.StringValue(remote.CreateTime); !got.Equal(want) {
		t.Errorf("create_time = %s, want %s", got, want)
	}
	if got, want := state.UpdateTime, types.StringValue(updateTime); !got.Equal(want) {
		t.Errorf("update_time = %s, want %s", got, want)
	}

	tags, tagDiagnostics := types.SetValueFrom(t.Context(), types.StringType, remote.Tags)
	if tagDiagnostics.HasError() {
		t.Fatalf("create expected tags: %v", tagDiagnostics)
	}
	if !state.Tags.Equal(tags) {
		t.Errorf("tags = %s, want %s", state.Tags, tags)
	}
	assertDynamicJSONObject(t, "test_set_metadata", state.TestSetMetadata, remote.TestSetMetadata)
	assertDynamicJSONObject(t, "parameters", state.Parameters, remote.Parameters)
}

func TestTestSetStateMapsNullDescriptionByConsumer(t *testing.T) {
	t.Parallel()

	remote := client.TestSet{
		Name:            "test-sets/abc12345",
		ID:              "abc12345",
		Slug:            "provider-test",
		DisplayName:     "Provider test",
		TestSetMetadata: json.RawMessage(`{}`),
		Parameters:      json.RawMessage(`{}`),
		Tags:            []string{},
		CreateTime:      "2026-09-15T00:00:00Z",
	}

	resourceState, resourceDiagnostics := testSetResourceState(t.Context(), remote, nil)
	if resourceDiagnostics.HasError() {
		t.Fatalf("testSetResourceState(): %v", resourceDiagnostics)
	}
	if got, want := resourceState.Description, types.StringValue(""); !got.Equal(want) {
		t.Errorf("resource description = %s, want %s", got, want)
	}

	dataSourceState, dataSourceDiagnostics := testSetDataSourceState(t.Context(), remote, nil)
	if dataSourceDiagnostics.HasError() {
		t.Fatalf("testSetDataSourceState(): %v", dataSourceDiagnostics)
	}
	if !dataSourceState.Description.IsNull() {
		t.Errorf("data-source description = %s, want null", dataSourceState.Description)
	}
}

func assertDynamicJSONObject(t *testing.T, name string, value types.Dynamic, want json.RawMessage) {
	t.Helper()

	got, err := dynamicJSONObject(value)
	if err != nil {
		t.Fatalf("encode %s: %v", name, err)
	}
	if got == nil || !jsonValuesEqual(*got, want) {
		t.Errorf("%s = %s, want %s", name, got, want)
	}
}
