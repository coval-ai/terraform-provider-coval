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
		t.Fatalf("metadata = %s", json.RawMessage(*input.Metadata))
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
