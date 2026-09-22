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

func TestRunTemplateResourceSchemaMatchesPublicContract(t *testing.T) {
	t.Parallel()

	var response resource.SchemaResponse
	newRunTemplateResource().Schema(t.Context(), resource.SchemaRequest{}, &response)
	for _, name := range []string{"agent_ids", "persona_ids", "test_set_ids"} {
		attribute, ok := response.Schema.Attributes[name].(schema.SetAttribute)
		if !ok || !attribute.Required {
			t.Errorf("%s must be a required set", name)
		}
	}
	for _, name := range []string{"metric_ids", "mutation_ids", "tags"} {
		attribute, ok := response.Schema.Attributes[name].(schema.SetAttribute)
		if !ok || !attribute.Optional || !attribute.Computed {
			t.Errorf("%s must be an optional, computed set", name)
		}
	}
	seed, ok := response.Schema.Attributes["sub_sample_seed"].(schema.Int64Attribute)
	if !ok || !seed.Optional || seed.Computed {
		t.Errorf("sub_sample_seed must be optional and caller-controlled: %#v", seed)
	}
	for _, obsolete := range []string{"agent_id", "persona_id", "test_set_id"} {
		if _, ok := response.Schema.Attributes[obsolete]; ok {
			t.Errorf("schema contains obsolete public field %q", obsolete)
		}
	}
}

func TestRunTemplateInputAndStateRoundTrip(t *testing.T) {
	t.Parallel()

	metadataObject := types.ObjectValueMust(
		map[string]attr.Type{"owner": types.StringType},
		map[string]attr.Value{"owner": types.StringValue("platform")},
	)
	plan := runTemplateResourceModel{
		WorkspaceID:    types.StringValue("workspace-1"),
		DisplayName:    types.StringValue("Nightly"),
		Description:    types.StringValue("Nightly support evaluation"),
		AgentIDs:       stringSetValue("abc123def456ghi789jklm"),
		PersonaIDs:     stringSetValue("def456ghi789jklmabc123"),
		TestSetIDs:     stringSetValue("abc12345"),
		MetricIDs:      types.SetValueMust(types.StringType, []attr.Value{}),
		MutationIDs:    types.SetValueMust(types.StringType, []attr.Value{}),
		IterationCount: types.Int64Value(2),
		Concurrency:    types.Int64Value(3),
		SubSampleSize:  types.Int64Value(4),
		SubSampleSeed:  types.Int64Value(5),
		Metadata:       types.DynamicValue(metadataObject),
		Tags:           stringSetValue("terraform", "nightly"),
	}
	input, diagnostics := createRunTemplateInput(t.Context(), plan)
	if diagnostics.HasError() {
		t.Fatalf("createRunTemplateInput() diagnostics: %v", diagnostics)
	}
	if len(input.AgentIDs) != 1 || len(input.PersonaIDs) != 1 || len(input.TestSetIDs) != 1 || input.SubSampleSeed == nil || *input.SubSampleSeed != 5 {
		t.Fatalf("input = %#v", input)
	}
	if input.Metadata == nil || string(*input.Metadata) != `{"owner":"platform"}` {
		t.Fatalf("metadata = %v", input.Metadata)
	}

	updatedPlan := plan
	updatedPlan.SubSampleSeed = types.Int64Null()
	update, diagnostics := updateRunTemplateInput(t.Context(), updatedPlan)
	if diagnostics.HasError() || update.SubSampleSeed != nil || update.MetricIDs == nil || update.Tags == nil {
		t.Fatalf("update = %#v, diagnostics = %v", update, diagnostics)
	}

	updateTime := "2026-09-22T01:00:00Z"
	creatorID := "ghi789jklmabc123def456"
	seedValue := int64(5)
	state, diagnostics := runTemplateResourceState(t.Context(), client.RunTemplate{
		Name: "runTemplates/abc123def456ghi789jklm", ID: "abc123def456ghi789jklm", DisplayName: "Nightly", Description: "Nightly support evaluation",
		AgentIDs: []string{"abc123def456ghi789jklm"}, PersonaIDs: []string{"def456ghi789jklmabc123"}, TestSetIDs: []string{"abc12345"},
		MetricIDs: []string{}, MutationIDs: []string{}, IterationCount: 2, Concurrency: 3, SubSampleSize: 4, SubSampleSeed: &seedValue,
		Metadata: json.RawMessage(`{"owner":"platform"}`), Tags: []string{"terraform", "nightly"},
		CreateTime: "2026-09-22T00:00:00Z", UpdateTime: &updateTime, CreatedByUserID: &creatorID,
	}, &plan)
	if diagnostics.HasError() {
		t.Fatalf("runTemplateResourceState() diagnostics: %v", diagnostics)
	}
	if state.WorkspaceID.ValueString() != "workspace-1" || state.SubSampleSeed.IsNull() || state.SubSampleSeed.ValueInt64() != 5 || state.Metadata.IsNull() {
		t.Fatalf("state = %#v", state)
	}
}

func stringSetValue(values ...string) types.Set {
	elements := make([]attr.Value, len(values))
	for index, value := range values {
		elements[index] = types.StringValue(value)
	}
	return types.SetValueMust(types.StringType, elements)
}
