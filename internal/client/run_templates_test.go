package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRunTemplateLifecycleRequests(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount++
		response.Header().Set("Content-Type", "application/json")
		switch requestCount {
		case 1:
			if request.Method != http.MethodPost || request.URL.Path != "/v1/run-templates" {
				t.Fatalf("create request = %s %s", request.Method, request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(body["agent_ids"], []any{"agent-1"}) || !reflect.DeepEqual(body["persona_ids"], []any{"persona-1"}) || !reflect.DeepEqual(body["test_set_ids"], []any{"tests-1"}) {
				t.Fatalf("create relationship fields = %#v", body)
			}
			for _, singular := range []string{"agent_id", "persona_id", "test_set_id"} {
				if _, ok := body[singular]; ok {
					t.Fatalf("create body contains obsolete %q: %#v", singular, body)
				}
			}
			_, _ = response.Write([]byte(runTemplateTestEnvelope))
		case 2:
			if request.Method != http.MethodGet || request.URL.Path != "/v1/run-templates/abc123def456ghi789jklm" {
				t.Fatalf("get request = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(runTemplateTestEnvelope))
		case 3:
			if request.Method != http.MethodPatch || request.URL.Path != "/v1/run-templates/abc123def456ghi789jklm" {
				t.Fatalf("update request = %s %s", request.Method, request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if seed, present := body["sub_sample_seed"]; !present || seed != nil {
				t.Fatalf("update sub_sample_seed = %#v, present = %t", seed, present)
			}
			if !reflect.DeepEqual(body["metric_ids"], []any{}) || !reflect.DeepEqual(body["tags"], []any{}) {
				t.Fatalf("update clearable fields = %#v", body)
			}
			_, _ = response.Write([]byte(runTemplateTestEnvelope))
		case 4:
			if request.Method != http.MethodDelete || request.URL.Path != "/v1/run-templates/abc123def456ghi789jklm" {
				t.Fatalf("delete request = %s %s", request.Method, request.URL.Path)
			}
			response.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	}))
	defer server.Close()

	apiClient, err := New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	description := "Nightly support evaluation"
	input := CreateRunTemplateInput{
		DisplayName: "Nightly", Description: &description,
		AgentIDs: []string{"agent-1"}, PersonaIDs: []string{"persona-1"}, TestSetIDs: []string{"tests-1"},
		IterationCount: 1, Concurrency: 1, SubSampleSize: 0,
	}
	created, err := apiClient.CreateRunTemplate(t.Context(), input)
	if err != nil || created.ID == "" {
		t.Fatalf("CreateRunTemplate() = %#v, %v", created, err)
	}
	if _, err := apiClient.GetRunTemplate(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.UpdateRunTemplate(t.Context(), created.ID, UpdateRunTemplateInput{
		DisplayName: "Nightly", Description: description,
		AgentIDs: []string{"agent-1"}, PersonaIDs: []string{"persona-1"}, TestSetIDs: []string{"tests-1"},
		MetricIDs: []string{}, MutationIDs: []string{}, IterationCount: 1, Concurrency: 1, SubSampleSize: 0, Tags: []string{},
	}); err != nil {
		t.Fatal(err)
	}
	if err := apiClient.DeleteRunTemplate(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
}

func TestListRunTemplatesEncodesOptions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if request.Method != http.MethodGet || request.URL.Path != "/v1/run-templates" || query.Get("page_size") != "100" || query.Get("page_token") != "next" {
			t.Fatalf("request = %s %s?%s", request.Method, request.URL.Path, request.URL.RawQuery)
		}
		if got := query["tag_filters"]; !reflect.DeepEqual(got, []string{"terraform", "nightly"}) {
			t.Fatalf("tag_filters = %#v", got)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"run_templates":[],"next_page_token":""}`))
	}))
	defer server.Close()

	apiClient, err := New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.ListRunTemplates(t.Context(), ListRunTemplatesOptions{PageSize: 100, PageToken: "next", TagFilters: []string{"terraform", "nightly"}}); err != nil {
		t.Fatal(err)
	}
}

const runTemplateTestEnvelope = `{"run_template":{"name":"runTemplates/abc123def456ghi789jklm","id":"abc123def456ghi789jklm","display_name":"Nightly","description":"Nightly support evaluation","agent_ids":["agent-1"],"persona_ids":["persona-1"],"test_set_ids":["tests-1"],"metric_ids":[],"mutation_ids":[],"iteration_count":1,"concurrency":1,"sub_sample_size":0,"sub_sample_seed":null,"metadata":{},"tags":[],"create_time":"2026-09-22T00:00:00Z","update_time":null,"created_by_user_id":null}}`
