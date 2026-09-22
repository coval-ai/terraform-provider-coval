package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
)

func TestListAllTestSetsPaginates(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount++
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page_token") {
		case "":
			_, _ = fmt.Fprint(response, `{"test_sets":[{"id":"abc12345"}],"next_page_token":"page-2"}`)
		case "page-2":
			_, _ = fmt.Fprint(response, `{"test_sets":[{"id":"def67890"}],"next_page_token":""}`)
		default:
			t.Errorf("unexpected page token %q", request.URL.Query().Get("page_token"))
		}
	}))
	defer server.Close()

	apiClient, err := client.New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatalf("client.New(): %v", err)
	}
	results, err := listAllTestSets(t.Context(), apiClient, client.ListTestSetsOptions{PageSize: 100})
	if err != nil {
		t.Fatalf("listAllTestSets(): %v", err)
	}
	if requestCount != 2 || len(results) != 2 || results[1].ID != "def67890" {
		t.Fatalf("requests = %d, results = %#v", requestCount, results)
	}
}

func TestListAllTestCasesRejectsRepeatedToken(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		requestCount++
		response.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(response, `{"test_cases":[],"next_page_token":"same-token"}`)
	}))
	defer server.Close()

	apiClient, err := client.New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatalf("client.New(): %v", err)
	}
	_, err = listAllTestCases(t.Context(), apiClient, client.ListTestCasesOptions{PageSize: 100})
	if err == nil {
		t.Fatal("listAllTestCases() accepted a repeated pagination token")
	}
	if requestCount != 2 {
		t.Fatalf("requests = %d, want 2", requestCount)
	}
}

func TestListAllPersonasPaginates(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount++
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page_token") {
		case "":
			_, _ = fmt.Fprint(response, `{"personas":[{"id":"abc123def456ghi789jklm"}],"next_page_token":"page-2"}`)
		case "page-2":
			_, _ = fmt.Fprint(response, `{"personas":[{"id":"def456ghi789jklmabc123"}],"next_page_token":""}`)
		default:
			t.Errorf("unexpected page token %q", request.URL.Query().Get("page_token"))
		}
	}))
	defer server.Close()

	apiClient, err := client.New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatalf("client.New(): %v", err)
	}
	results, err := listAllPersonas(t.Context(), apiClient, client.ListPersonasOptions{PageSize: 100})
	if err != nil {
		t.Fatalf("listAllPersonas(): %v", err)
	}
	if requestCount != 2 || len(results) != 2 || results[1].ID != "def456ghi789jklmabc123" {
		t.Fatalf("requests = %d, results = %#v", requestCount, results)
	}
}

func TestListAllScheduledRunsPaginates(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page_token") {
		case "":
			_, _ = fmt.Fprint(response, `{"scheduled_runs":[{"id":"xyz789uvw456rst123abcd"}],"next_page_token":"page-2"}`)
		case "page-2":
			_, _ = fmt.Fprint(response, `{"scheduled_runs":[{"id":"abc123def456ghi789jklm"}],"next_page_token":null}`)
		default:
			t.Fatalf("unexpected token")
		}
	}))
	defer server.Close()
	apiClient, err := client.New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	results, err := listAllScheduledRuns(t.Context(), apiClient, client.ListScheduledRunsOptions{PageSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || len(results) != 2 {
		t.Fatalf("requests = %d, results = %#v", requests, results)
	}
}

func TestListAllAlertsPaginates(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount++
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page_token") {
		case "":
			_, _ = fmt.Fprint(response, `{"alerts":[{"ulid":"01HZ0EXAMPLE00000000000000"}],"next_page_token":"page-2","total_count":2}`)
		case "page-2":
			_, _ = fmt.Fprint(response, `{"alerts":[{"ulid":"01HZ0EXAMPLE00000000000001"}],"next_page_token":null,"total_count":2}`)
		default:
			t.Errorf("unexpected page token %q", request.URL.Query().Get("page_token"))
		}
	}))
	defer server.Close()
	apiClient, err := client.New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	results, err := listAllAlerts(t.Context(), apiClient, client.ListAlertsOptions{PageSize: 100})
	if err != nil {
		t.Fatalf("listAllAlerts(): %v", err)
	}
	if requestCount != 2 || len(results) != 2 {
		t.Fatalf("requests = %d, results = %#v", requestCount, results)
	}
}

func TestListAllRunTemplatesPaginates(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestCount++
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Query().Get("page_token") {
		case "":
			_, _ = fmt.Fprint(response, `{"run_templates":[{"id":"abc123def456ghi789jklm"}],"next_page_token":"page-2"}`)
		case "page-2":
			_, _ = fmt.Fprint(response, `{"run_templates":[{"id":"def456ghi789jklmabc123"}],"next_page_token":""}`)
		default:
			t.Errorf("unexpected page token %q", request.URL.Query().Get("page_token"))
		}
	}))
	defer server.Close()

	apiClient, err := client.New("test-key", server.URL+"/v1")
	if err != nil {
		t.Fatalf("client.New(): %v", err)
	}
	results, err := listAllRunTemplates(t.Context(), apiClient, client.ListRunTemplatesOptions{PageSize: 100})
	if err != nil {
		t.Fatalf("listAllRunTemplates(): %v", err)
	}
	if requestCount != 2 || len(results) != 2 || results[1].ID != "def456ghi789jklmabc123" {
		t.Fatalf("requests = %d, results = %#v", requestCount, results)
	}
}
