package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestCreateTestCase(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/test-cases" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if decoded["test_set_id"] != "abc12345" || decoded["input_str"] != "Ask about the refund policy" {
			t.Errorf("request body = %#v", decoded)
		}
		behaviors, ok := decoded["expected_behaviors"].([]any)
		if !ok || len(behaviors) != 2 {
			t.Errorf("expected_behaviors = %#v", decoded["expected_behaviors"])
		}
		if metadata, ok := decoded["simulation_metadata_input"].(map[string]any); !ok || metadata["channel"] != "voice" {
			t.Errorf("simulation_metadata_input = %#v", decoded["simulation_metadata_input"])
		}
		if metric, ok := decoded["metric_input"].(map[string]any); !ok || metric["policy_window_days"] != float64(30) {
			t.Errorf("metric_input = %#v", decoded["metric_input"])
		}
		return testResponse(http.StatusCreated, `{"test_case":{"name":"test-cases/abc123def456ghi789jklm","id":"abc123def456ghi789jklm","test_set_id":"abc12345","input_str":"Ask about the refund policy","expected_behaviors":["Explain the policy","Offer next steps"],"expected_output_json":{"eligible":true},"description":"Refund policy","input_type":"SCENARIO","script_turns":null,"simulation_metadata_input":{"channel":"voice"},"metric_input":{"policy_window_days":30},"user_notes":"Regression","create_time":"2026-09-14T00:00:00Z","update_time":null}}`, nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	behaviors := []string{"Explain the policy", "Offer next steps"}
	expectedOutput := json.RawMessage(`{"eligible":true}`)
	simulationMetadata := json.RawMessage(`{"channel":"voice"}`)
	metricInput := json.RawMessage(`{"policy_window_days":30}`)
	description := "Refund policy"
	inputType := "SCENARIO"
	userNotes := "Regression"
	created, err := apiClient.CreateTestCase(context.Background(), CreateTestCaseInput{
		TestSetID:          "abc12345",
		InputString:        "Ask about the refund policy",
		ExpectedBehaviors:  &behaviors,
		ExpectedOutputJSON: &expectedOutput,
		Description:        &description,
		InputType:          &inputType,
		SimulationMetadata: &simulationMetadata,
		MetricInput:        &metricInput,
		UserNotes:          &userNotes,
	})
	if err != nil {
		t.Fatalf("CreateTestCase(): %v", err)
	}
	if created.ID != "abc123def456ghi789jklm" || created.TestSetID == nil || *created.TestSetID != "abc12345" {
		t.Errorf("created = %#v", created)
	}
	if string(created.MetricInput) != string(metricInput) {
		t.Errorf("metric_input = %s", created.MetricInput)
	}
}

func TestGetAndUpdateTestCase(t *testing.T) {
	t.Parallel()

	requestCount := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount++
		if request.URL.Path != "/v1/test-cases/abc123def456ghi789jklm" {
			t.Fatalf("request path = %s", request.URL.Path)
		}
		switch requestCount {
		case 1:
			if request.Method != http.MethodGet {
				t.Fatalf("request method = %s, want GET", request.Method)
			}
			return testResponse(http.StatusOK, `{"test_case":{"name":"test-cases/abc123def456ghi789jklm","id":"abc123def456ghi789jklm","test_set_id":"abc12345","input_str":"Original","expected_behaviors":[],"expected_output_json":{},"description":null,"input_type":"SCENARIO","script_turns":null,"simulation_metadata_input":{},"metric_input":{},"user_notes":null,"create_time":"2026-09-14T00:00:00Z","update_time":null}}`, nil), nil
		case 2:
			if request.Method != http.MethodPatch {
				t.Fatalf("request method = %s, want PATCH", request.Method)
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if decoded["test_set_id"] != "def67890" || decoded["input_str"] != "Updated" {
				t.Errorf("request body = %#v", decoded)
			}
			if behaviors, ok := decoded["expected_behaviors"].([]any); !ok || len(behaviors) != 0 {
				t.Errorf("expected_behaviors = %#v", decoded["expected_behaviors"])
			}
			return testResponse(http.StatusOK, `{"test_case":{"name":"test-cases/abc123def456ghi789jklm","id":"abc123def456ghi789jklm","test_set_id":"def67890","input_str":"Updated","expected_behaviors":[],"expected_output_json":{},"description":"","input_type":"SCRIPT","script_turns":[{"type":"skip"}],"simulation_metadata_input":{},"metric_input":{},"user_notes":"","create_time":"2026-09-14T00:00:00Z","update_time":"2026-09-14T01:00:00Z"}}`, nil), nil
		default:
			t.Fatalf("unexpected request %d", requestCount)
			return nil, nil
		}
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if _, err := apiClient.GetTestCase(context.Background(), "abc123def456ghi789jklm"); err != nil {
		t.Fatalf("GetTestCase(): %v", err)
	}
	testSetID := "def67890"
	inputString := "Updated"
	behaviors := []string{}
	description := ""
	inputType := "SCRIPT"
	scriptTurns := json.RawMessage(`[{"type":"skip"}]`)
	userNotes := ""
	updated, err := apiClient.UpdateTestCase(context.Background(), "abc123def456ghi789jklm", UpdateTestCaseInput{
		TestSetID:         &testSetID,
		InputString:       &inputString,
		ExpectedBehaviors: &behaviors,
		Description:       &description,
		InputType:         &inputType,
		ScriptTurns:       &scriptTurns,
		UserNotes:         &userNotes,
	})
	if err != nil {
		t.Fatalf("UpdateTestCase(): %v", err)
	}
	if updated.TestSetID == nil || *updated.TestSetID != "def67890" || updated.InputType == nil || *updated.InputType != "SCRIPT" {
		t.Errorf("updated = %#v", updated)
	}
}

func TestListTestCasesEncodesOptions(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/v1/test-cases" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("filter") != `test_set_id="abc12345"` || query.Get("page_size") != "100" || query.Get("page_token") != "next" || query.Get("order_by") != "-create_time" {
			t.Errorf("query = %v", query)
		}
		return testResponse(http.StatusOK, `{"test_cases":[],"next_page_token":""}`, nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	_, err = apiClient.ListTestCases(context.Background(), ListTestCasesOptions{
		Filter:    `test_set_id="abc12345"`,
		PageSize:  100,
		PageToken: "next",
		OrderBy:   "-create_time",
	})
	if err != nil {
		t.Fatalf("ListTestCases(): %v", err)
	}
}

func TestDeleteTestCase(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodDelete || request.URL.Path != "/v1/test-cases/abc123def456ghi789jklm" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		return testResponse(http.StatusOK, `{}`, nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := apiClient.DeleteTestCase(context.Background(), "abc123def456ghi789jklm"); err != nil {
		t.Fatalf("DeleteTestCase(): %v", err)
	}
}
