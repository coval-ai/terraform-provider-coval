package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScheduledRunLifecycleRequests(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			if request.Method != http.MethodPost || request.URL.Path != "/v1/scheduled-runs" {
				t.Fatalf("create = %s %s", request.Method, request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["enabled"] != false || body["schedule_expression"] != "rate(1 day)" {
				t.Fatalf("body = %#v", body)
			}
			_, _ = response.Write([]byte(scheduledRunTestEnvelope))
		case 2:
			if request.Method != http.MethodGet || request.URL.Path != "/v1/scheduled-runs/xyz789uvw456rst123abcd" {
				t.Fatalf("get = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(scheduledRunTestEnvelope))
		case 3:
			if request.Method != http.MethodPatch {
				t.Fatalf("update = %s", request.Method)
			}
			_, _ = response.Write([]byte(scheduledRunTestEnvelope))
		case 4:
			if request.Method != http.MethodDelete {
				t.Fatalf("delete = %s", request.Method)
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
	input := CreateScheduledRunInput{DisplayName: "Nightly", RunTemplateID: "abc123def456ghi789jklm", ScheduleExpression: "rate(1 day)", ScheduleTimezone: "UTC", Enabled: false}
	created, err := apiClient.CreateScheduledRun(t.Context(), input)
	if err != nil || created.ID == "" {
		t.Fatalf("CreateScheduledRun() = %#v, %v", created, err)
	}
	if _, err := apiClient.GetScheduledRun(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.UpdateScheduledRun(t.Context(), created.ID, input); err != nil {
		t.Fatal(err)
	}
	if err := apiClient.DeleteScheduledRun(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
}

func TestListScheduledRunsEncodesFilters(t *testing.T) {
	t.Parallel()
	enabled := false
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if query.Get("enabled") != "false" || query.Get("template_id") != "abc123def456ghi789jklm" || query.Get("page_size") != "100" || query.Get("page_token") != "next" {
			t.Fatalf("query = %v", query)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"scheduled_runs":[],"next_page_token":null,"total_count":0}`))
	}))
	defer server.Close()
	apiClient, err := New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.ListScheduledRuns(t.Context(), ListScheduledRunsOptions{Enabled: &enabled, RunTemplateID: "abc123def456ghi789jklm", PageSize: 100, PageToken: "next"}); err != nil {
		t.Fatal(err)
	}
}

const scheduledRunTestEnvelope = `{"scheduled_run":{"name":"scheduled-runs/xyz789uvw456rst123abcd","id":"xyz789uvw456rst123abcd","display_name":"Nightly","run_template_id":"abc123def456ghi789jklm","schedule_expression":"rate(1 day)","schedule_timezone":"UTC","enabled":false,"last_run_at":null,"last_run_id":null,"create_time":"2026-09-22T00:00:00Z","update_time":null}}`
