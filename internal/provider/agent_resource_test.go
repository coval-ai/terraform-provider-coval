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

func TestAgentResourceSchemaAndInput(t *testing.T) {
	t.Parallel()

	var response resource.SchemaResponse
	(&agentResource{}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	for _, name := range []string{"display_name", "model_type"} {
		attribute, ok := response.Schema.Attributes[name].(schema.StringAttribute)
		if !ok || !attribute.Required {
			t.Errorf("%s must be a required string", name)
		}
	}
	metadataAttribute, ok := response.Schema.Attributes["metadata"].(schema.DynamicAttribute)
	if !ok || !metadataAttribute.Sensitive {
		t.Error("metadata must be sensitive")
	}
	attributesAttribute, ok := response.Schema.Attributes["attributes"].(schema.DynamicAttribute)
	if !ok || !attributesAttribute.Optional || !attributesAttribute.Computed || len(attributesAttribute.PlanModifiers) == 0 {
		t.Error("attributes must be optional, computed, and preserve state while unknown")
	}
	knowledgeBaseIDsAttribute, ok := response.Schema.Attributes["knowledge_base_ids"].(schema.SetAttribute)
	if !ok || !knowledgeBaseIDsAttribute.Computed || len(knowledgeBaseIDsAttribute.PlanModifiers) == 0 {
		t.Error("knowledge_base_ids must be computed and preserve state while unknown")
	}

	metadata := types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"chat_endpoint": types.StringType},
		map[string]attr.Value{"chat_endpoint": types.StringValue("https://example.com/chat")},
	))
	plan := agentResourceModel{
		DisplayName: types.StringValue("Support"),
		ModelType:   types.StringValue("MODEL_TYPE_CHAT"),
		Metadata:    metadata,
		Attributes:  types.DynamicNull(),
		Workflows:   types.DynamicNull(),
		MetricIDs:   types.SetNull(types.StringType),
		TestSetIDs:  types.SetNull(types.StringType),
		Tags:        types.SetNull(types.StringType),
	}
	input, diagnostics := createAgentInput(context.Background(), plan)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if input.Metadata == nil {
		t.Fatal("metadata is nil")
	}
	if string(*input.Metadata) != `{"chat_endpoint":"https://example.com/chat"}` {
		t.Fatalf("metadata = %s", *input.Metadata)
	}
}

func TestAgentStatePreservesNullableAndJSONFields(t *testing.T) {
	t.Parallel()

	state, diagnostics := agentResourceState(context.Background(), client.Agent{
		ID:               "abc123def456ghi789jklm",
		CustomerAgentID:  "support",
		DisplayName:      "Support",
		ModelType:        "MODEL_TYPE_CHAT",
		Attributes:       json.RawMessage(`null`),
		Metadata:         json.RawMessage(`{"chat_endpoint":"https://example.com/chat"}`),
		Workflows:        json.RawMessage(`{}`),
		MetricIDs:        []string{},
		TestSetIDs:       []string{},
		KnowledgeBaseIDs: []string{},
		Tags:             []string{},
		CreateTime:       "2026-09-18T00:00:00Z",
	}, nil)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if !state.Attributes.IsNull() || state.Metadata.IsNull() || !state.PhoneNumber.IsNull() {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestAgentStateUsesRemoteMetadataWhenAPIAddsFields(t *testing.T) {
	t.Parallel()

	configured := types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"chat_endpoint": types.StringType},
		map[string]attr.Value{"chat_endpoint": types.StringValue("https://example.com/chat")},
	))
	prior := &agentResourceModel{Metadata: configured}
	remoteMetadata := json.RawMessage(`{
		"chat_endpoint":"https://example.com/chat",
		"custom_headers":{"X-Managed-Outside-Terraform":"true"}
	}`)
	state, diagnostics := agentResourceState(context.Background(), client.Agent{
		ID:              "abc123def456ghi789jklm",
		CustomerAgentID: "support",
		DisplayName:     "Support",
		ModelType:       "MODEL_TYPE_CHAT_A2A",
		Metadata:        remoteMetadata,
		Workflows:       json.RawMessage(`{}`),
		CreateTime:      "2026-09-18T00:00:00Z",
	}, prior)
	if diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if state.Metadata.Equal(configured) {
		t.Fatal("metadata hid fields added outside Terraform")
	}
	metadata, err := dynamicJSONObject(state.Metadata)
	if err != nil {
		t.Fatal(err)
	}
	if metadata == nil || !jsonValuesEqual(*metadata, remoteMetadata) {
		t.Fatalf("metadata = %s", metadata)
	}
}

func TestAgentStateAfterMutationPreservesPlannedMetadataWhenAPIAddsFields(t *testing.T) {
	t.Parallel()

	planned := types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"chat_endpoint": types.StringType},
		map[string]attr.Value{"chat_endpoint": types.StringValue("https://example.com/chat")},
	))
	remote := client.Agent{
		ID:              "abc123def456ghi789jklm",
		CustomerAgentID: "support",
		DisplayName:     "Support",
		ModelType:       "MODEL_TYPE_CHAT_A2A",
		Metadata:        json.RawMessage(`{"chat_endpoint":"https://example.com/chat","response_message_path":"result.text"}`),
		Workflows:       json.RawMessage(`{}`),
		CreateTime:      "2026-09-18T00:00:00Z",
	}

	state, diagnostics := agentResourceStateAfterMutation(context.Background(), remote, &agentResourceModel{Metadata: planned})
	if diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if !state.Metadata.Equal(planned) {
		t.Fatalf("metadata = %#v, want planned value %#v", state.Metadata, planned)
	}

	serverAdditions, ok := agentMetadataServerAdditions(remote.Metadata, planned)
	if !ok {
		t.Fatal("metadata additions were not detected")
	}
	normalizedMetadata, err := jsonObjectWithoutMatchingAdditions(remote.Metadata, serverAdditions)
	if err != nil {
		t.Fatal(err)
	}
	remote.Metadata = normalizedMetadata
	refreshed, diagnostics := agentResourceState(context.Background(), remote, &state)
	if diagnostics.HasError() {
		t.Fatalf("refresh diagnostics = %v", diagnostics)
	}
	if !refreshed.Metadata.Equal(planned) {
		t.Fatalf("metadata after refresh = %#v, want planned value %#v", refreshed.Metadata, planned)
	}
}

func TestAgentStateAfterMutationUsesRemoteMetadataWhenConfiguredValueChanges(t *testing.T) {
	t.Parallel()

	configured := types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"chat_endpoint": types.StringType},
		map[string]attr.Value{"chat_endpoint": types.StringValue("https://example.com/old")},
	))
	state, diagnostics := agentResourceStateAfterMutation(context.Background(), client.Agent{
		ID:              "abc123def456ghi789jklm",
		CustomerAgentID: "support",
		DisplayName:     "Support",
		ModelType:       "MODEL_TYPE_CHAT",
		Metadata:        json.RawMessage(`{"chat_endpoint":"https://example.com/new","api_default":true}`),
		Workflows:       json.RawMessage(`{}`),
		CreateTime:      "2026-09-18T00:00:00Z",
	}, &agentResourceModel{Metadata: configured})
	if diagnostics.HasError() {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if state.Metadata.Equal(configured) {
		t.Fatal("metadata preserved a configured value that the API changed")
	}
	if _, ok := agentMetadataServerAdditions(json.RawMessage(`{"chat_endpoint":"https://example.com/new","api_default":true}`), configured); ok {
		t.Fatal("changed configured metadata was classified as a server addition")
	}
}

func TestListAllAgentsRejectsRepeatedToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"agents":[],"next_page_token":"same-token"}`)
	}))
	defer server.Close()

	apiClient, err := client.New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := listAllAgents(t.Context(), apiClient, client.ListAgentsOptions{PageSize: 100}); err == nil {
		t.Fatal("listAllAgents accepted a repeated token")
	}
}
