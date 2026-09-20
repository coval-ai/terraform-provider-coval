package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMetricLifecycleRequests(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.Header().Set("Content-Type", "application/json")
		switch requests {
		case 1:
			if request.Method != http.MethodPost || request.URL.Path != "/v1/metrics" {
				t.Fatalf("create request = %s %s", request.Method, request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["metric_name"] != "Resolution" || body["metric_type"] != "METRIC_LLM_BINARY" {
				t.Fatalf("create body = %#v", body)
			}
			_, _ = response.Write([]byte(metricTestEnvelope))
		case 2:
			if request.Method != http.MethodGet || request.URL.Path != "/v1/metrics/abc123def456ghi789jklm" {
				t.Fatalf("get request = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(metricTestEnvelope))
		case 3:
			if request.Method != http.MethodPatch || request.URL.Path != "/v1/metrics/abc123def456ghi789jklm" {
				t.Fatalf("update request = %s %s", request.Method, request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if runtimeConfig, ok := body["runtime_config"]; !ok || runtimeConfig != nil {
				t.Fatalf("update runtime_config = %#v, present = %t", runtimeConfig, ok)
			}
			_, _ = response.Write([]byte(metricTestEnvelope))
		case 4:
			if request.Method != http.MethodDelete {
				t.Fatalf("delete method = %s", request.Method)
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
	input := CreateMetricInput{MetricName: "Resolution", Description: "Whether the issue was resolved", MetricType: "METRIC_LLM_BINARY"}
	created, err := apiClient.CreateMetric(t.Context(), input)
	if err != nil || created.ID == "" {
		t.Fatalf("CreateMetric() = %#v, %v", created, err)
	}
	if _, err := apiClient.GetMetric(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := apiClient.UpdateMetric(t.Context(), created.ID, UpdateMetricInput{CreateMetricInput: input, ClearRuntimeConfig: true}); err != nil {
		t.Fatal(err)
	}
	if err := apiClient.DeleteMetric(t.Context(), created.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSQLMetricAggregationAndUnitRequests(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["aggregation_method"] != "SUM" || body["unit"] != "count" {
			t.Fatalf("%s body = %#v", request.Method, body)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"metric":{"id":"example-metric","aggregation_method":"SUM","unit":"count"}}`))
	}))
	defer server.Close()
	apiClient, err := New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	method, unit := "SUM", "count"
	input := CreateMetricInput{MetricName: "Question Count", Description: "Counts questions", MetricType: "METRIC_SQL_FLOAT", AggregationMethod: &method, Unit: &unit}
	created, err := apiClient.CreateMetric(t.Context(), input)
	if err != nil || created.AggregationMethod == nil || *created.AggregationMethod != method || created.Unit == nil || *created.Unit != unit {
		t.Fatalf("CreateMetric() = %#v, %v", created, err)
	}
	updated, err := apiClient.UpdateMetric(t.Context(), created.ID, UpdateMetricInput{CreateMetricInput: input})
	if err != nil || updated.AggregationMethod == nil || *updated.AggregationMethod != method || updated.Unit == nil || *updated.Unit != unit {
		t.Fatalf("UpdateMetric() = %#v, %v", updated, err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestListMetricsEncodesPublicFilters(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		if query.Get("filter") != "metric_type=METRIC_LLM_BINARY" || query.Get("include_builtin") != "true" || query.Get("page_size") != "100" {
			t.Fatalf("query = %v", query)
		}
		if got := query["tag_filters"]; !reflect.DeepEqual(got, []string{"production", "llm"}) {
			t.Fatalf("tag_filters = %#v", got)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"metrics":[],"next_page_token":""}`))
	}))
	defer server.Close()
	apiClient, err := New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = apiClient.ListMetrics(t.Context(), ListMetricsOptions{Filter: "metric_type=METRIC_LLM_BINARY", PageSize: 100, IncludeBuiltin: true, TagFilters: []string{"production", "llm"}})
	if err != nil {
		t.Fatal(err)
	}
}

const metricTestEnvelope = `{"metric":{"name":"metrics/abc123def456ghi789jklm","id":"abc123def456ghi789jklm","metric_name":"Resolution","description":"Whether the issue was resolved","metric_type":"METRIC_LLM_BINARY","evaluation":null,"prompt":"Did the agent resolve the issue?","enabled_tools":null,"categories":null,"min_value":null,"max_value":null,"metadata_field_type":null,"metadata_field_key":null,"regex_pattern":null,"role":null,"min_pause_duration_seconds":null,"max_silence_duration_seconds":null,"min_silence_gap_seconds":null,"frequency_threshold":null,"direction":null,"success_sentiments":null,"percent_above":null,"success_end_reasons":null,"observation_name":null,"expected_body":null,"match_path":null,"min_volume_change_for_pitch_misalignment":null,"threshold":null,"operator":null,"ivr_flow":null,"sql_query":null,"criteria_source":null,"criteria_path":null,"criteria":null,"reporting_method":null,"base_prompt_template":null,"include_traces":null,"runtime_config":null,"target_condition":null,"tags":[],"created_by":null,"create_time":"2026-09-18T00:00:00Z","update_time":null,"current_version":null}}`
