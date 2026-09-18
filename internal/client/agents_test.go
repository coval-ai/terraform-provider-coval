package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestAgentLifecycleRequests(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			if request.Method != http.MethodPost || request.URL.Path != "/v1/agents" {
				t.Fatalf("create request = %s %s", request.Method, request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["display_name"] != "Support" || body["model_type"] != "MODEL_TYPE_CHAT" {
				t.Fatalf("create body = %#v", body)
			}
			_, _ = response.Write([]byte(agentTestEnvelope))
		case 2:
			if request.Method != http.MethodGet || request.URL.Path != "/v1/agents/abc123def456ghi789jklm" {
				t.Fatalf("get request = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(agentTestEnvelope))
		case 3:
			if request.Method != http.MethodPatch || request.URL.Path != "/v1/agents/abc123def456ghi789jklm" {
				t.Fatalf("update request = %s %s", request.Method, request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["phone_number"] != nil || body["attributes"] != nil {
				t.Fatalf("nullable fields were not cleared: %#v", body)
			}
			_, _ = response.Write([]byte(agentTestEnvelope))
		case 4:
			if request.Method != http.MethodDelete || request.URL.Path != "/v1/agents/abc123def456ghi789jklm" {
				t.Fatalf("delete request = %s %s", request.Method, request.URL.Path)
			}
			response.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	defer server.Close()
	apiClient, err := New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	created, err := apiClient.CreateAgent(t.Context(), CreateAgentInput{DisplayName: "Support", ModelType: "MODEL_TYPE_CHAT"})
	if err != nil || created.ID == "" {
		t.Fatalf("CreateAgent() = %#v, %v", created, err)
	}
	if _, err := apiClient.GetAgent(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.UpdateAgent(t.Context(), created.ID, UpdateAgentInput{DisplayName: "Support", ModelType: "MODEL_TYPE_CHAT", CustomerAgentID: "support", Attributes: json.RawMessage(`null`), Metadata: json.RawMessage(`{}`), Workflows: json.RawMessage(`{}`), MetricIDs: []string{}, TestSetIDs: []string{}, Tags: []string{}}); err != nil {
		t.Fatal(err)
	}
	if err := apiClient.DeleteAgent(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
}

func TestListAgentsEncodesPublicFilters(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if query.Get("filter") != `display_name="Support Agent"` || query.Get("page_size") != "100" || query.Get("page_token") != "next" || query.Get("order_by") != "-create_time" {
			t.Fatalf("query = %v", query)
		}
		if got := query["tag_filters"]; !reflect.DeepEqual(got, []string{"production", "voice"}) {
			t.Fatalf("tag_filters = %#v", got)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"agents":[],"next_page_token":""}`))
	}))
	defer server.Close()
	apiClient, err := New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.ListAgents(t.Context(), ListAgentsOptions{Filter: `display_name="Support Agent"`, PageSize: 100, PageToken: "next", OrderBy: "-create_time", TagFilters: []string{"production", "voice"}})
	if err != nil {
		t.Fatal(err)
	}
}

const agentTestEnvelope = `{"agent":{"id":"abc123def456ghi789jklm","customer_agent_id":"support","display_name":"Support","model_type":"MODEL_TYPE_CHAT","phone_number":null,"endpoint":null,"prompt":null,"language":null,"attributes":null,"metadata":{},"workflows":{},"metric_ids":[],"test_set_ids":[],"knowledge_base_ids":[],"tags":[],"create_time":"2026-09-18T00:00:00Z","update_time":null}}`
