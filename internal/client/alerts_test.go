package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAlertLifecycleRequests(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount++
		response.Header().Set("Content-Type", "application/json")
		switch requestCount {
		case 1:
			if request.Method != http.MethodPost || request.URL.Path != "/v1/alerts" {
				t.Fatalf("create request = %s %s", request.Method, request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			conditions, ok := body["conditions"].([]any)
			if body["name"] != "Low Resolution" || !ok || len(conditions) != 1 {
				t.Fatalf("create body = %#v", body)
			}
			_, _ = response.Write([]byte(alertTestResponse))
		case 2:
			if request.Method != http.MethodGet || request.URL.Path != "/v1/alerts/01HZ0EXAMPLE00000000000000" {
				t.Fatalf("get request = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(alertTestResponse))
		case 3:
			if request.Method != http.MethodPatch || request.URL.Path != "/v1/alerts/01HZ0EXAMPLE00000000000000" {
				t.Fatalf("update request = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(alertTestResponse))
		case 4:
			if request.Method != http.MethodDelete {
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
	created, err := apiClient.CreateAlert(t.Context(), CreateAlertInput{Name: "Low Resolution", EvaluationType: "ON_RUN_COMPLETE", Conditions: json.RawMessage(`[{"aggregation":"RUN_AVERAGE","operator":"LT","threshold_float":0.9}]`)})
	if err != nil || created.ID == "" {
		t.Fatalf("CreateAlert() = %#v, %v", created, err)
	}
	if _, err := apiClient.GetAlert(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.UpdateAlert(t.Context(), created.ID, UpdateAlertInput{Name: "Low Resolution", EvaluationType: "ON_RUN_COMPLETE", ConversationSource: "ALL", MatchMode: "ALL", Conditions: json.RawMessage(`[{"aggregation":"RUN_AVERAGE","operator":"LT","threshold_float":0.9}]`), Channels: json.RawMessage(`[]`)}); err != nil {
		t.Fatal(err)
	}
	if err := apiClient.DeleteAlert(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
}

func TestListAlertsEncodesFilters(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if query.Get("conversation_source") != "SIMULATED" || query.Get("page_size") != "100" || query.Get("page_token") != "next" {
			t.Fatalf("query = %v", query)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"alerts":[],"next_page_token":null,"total_count":0}`))
	}))
	defer server.Close()
	apiClient, err := New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.ListAlerts(t.Context(), ListAlertsOptions{ConversationSource: "SIMULATED", PageSize: 100, PageToken: "next"}); err != nil {
		t.Fatal(err)
	}
}

const alertTestResponse = `{"ulid":"01HZ0EXAMPLE00000000000000","name":"Low Resolution","description":"","status":"ACTIVE","evaluation_type":"ON_RUN_COMPLETE","conversation_source":"ALL","match_mode":"ALL","cooldown_seconds":0,"custom_message_template":null,"agent_ids":[],"required_tags":[],"scheduled_run_ids":[],"trigger_count":0,"last_triggered_at":null,"conditions":[{"ulid":"01HZ0CONDITION000000000000","aggregation":"RUN_AVERAGE","operator":"LT","threshold_float":0.9}],"channels":[],"create_time":"2026-09-22T00:00:00Z","update_time":"2026-09-22T00:00:00Z"}`
