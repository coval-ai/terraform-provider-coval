package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

const testWorkspaceID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

func TestWorkspaceLifecycleRequests(t *testing.T) {
	t.Parallel()

	requestCount := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if request.Method != http.MethodPost || request.URL.Path != "/v1/workspaces" {
				t.Fatalf("create request = %s %s", request.Method, request.URL.Path)
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read create body: %v", err)
			}
			var input map[string]any
			if err := json.Unmarshal(body, &input); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if input["display_name"] != "Example Workspace" {
				t.Errorf("create input = %#v", input)
			}
			if _, exists := input["slug"]; exists {
				t.Errorf("create input includes deprecated slug: %#v", input)
			}
			return testResponse(http.StatusCreated, workspaceResponse("Example Workspace"), nil), nil
		case 2:
			if request.Method != http.MethodGet || request.URL.Path != "/v1/workspaces/"+testWorkspaceID {
				t.Fatalf("get request = %s %s", request.Method, request.URL.Path)
			}
			return testResponse(http.StatusOK, workspaceResponse("Example Workspace"), nil), nil
		case 3:
			if request.Method != http.MethodPatch || request.URL.Path != "/v1/workspaces/"+testWorkspaceID {
				t.Fatalf("update request = %s %s", request.Method, request.URL.Path)
			}
			return testResponse(http.StatusOK, workspaceResponse("Updated Example Workspace"), nil), nil
		case 4:
			if request.Method != http.MethodGet || request.URL.Path != "/v1/workspaces" {
				t.Fatalf("list request = %s %s", request.Method, request.URL.Path)
			}
			return testResponse(http.StatusOK, `{"workspaces":[`+workspaceJSON("Updated Example Workspace")+`]}`, nil), nil
		case 5:
			if request.Method != http.MethodDelete || request.URL.Path != "/v1/workspaces/"+testWorkspaceID {
				t.Fatalf("delete request = %s %s", request.Method, request.URL.Path)
			}
			return testResponse(http.StatusOK, workspaceResponse("Updated Example Workspace"), nil), nil
		default:
			t.Fatalf("unexpected request %d", requestCount)
			return nil, nil
		}
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	created, err := apiClient.CreateWorkspace(context.Background(), CreateWorkspaceInput{DisplayName: "Example Workspace"})
	if err != nil || created.DisplayName != "Example Workspace" {
		t.Fatalf("CreateWorkspace() = %#v, %v", created, err)
	}
	if _, err := apiClient.GetWorkspace(context.Background(), created.ID); err != nil {
		t.Fatalf("GetWorkspace(): %v", err)
	}
	updated, err := apiClient.UpdateWorkspace(context.Background(), created.ID, UpdateWorkspaceInput{DisplayName: "Updated Example Workspace"})
	if err != nil || updated.DisplayName != "Updated Example Workspace" {
		t.Fatalf("UpdateWorkspace() = %#v, %v", updated, err)
	}
	listed, err := apiClient.ListWorkspaces(context.Background())
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("ListWorkspaces() = %#v, %v", listed, err)
	}
	if err := apiClient.DeleteWorkspace(context.Background(), created.ID); err != nil {
		t.Fatalf("DeleteWorkspace(): %v", err)
	}
}

func workspaceResponse(displayName string) string {
	return `{"workspace":` + workspaceJSON(displayName) + `}`
}

func workspaceJSON(displayName string) string {
	return `{"id":"` + testWorkspaceID + `","slug":"example","display_name":"` + displayName + `","status":"ACTIVE","workspace_type":"CUSTOM","created_at":"2026-09-21T00:00:00Z","last_updated_at":"2026-09-21T01:00:00Z"}`
}
