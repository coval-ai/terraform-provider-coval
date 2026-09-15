package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestCreateTestSet(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/test-sets" {
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
		if decoded["display_name"] != "Provider test" {
			t.Errorf("display_name = %#v", decoded["display_name"])
		}
		metadata, ok := decoded["test_set_metadata"].(map[string]any)
		if !ok || metadata["owner"] != "terraform" {
			t.Errorf("test_set_metadata = %#v", decoded["test_set_metadata"])
		}
		return testResponse(http.StatusCreated, `{"test_set":{"name":"test-sets/abc12345","id":"abc12345","slug":"provider-test","display_name":"Provider test","description":"managed","test_set_type":"SCENARIO","test_set_metadata":{"owner":"terraform"},"parameters":{},"test_case_count":0,"tags":["ci"],"create_time":"2026-09-14T00:00:00Z","update_time":null}}`, nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	metadata := json.RawMessage(`{"owner":"terraform"}`)
	description := "managed"
	tags := []string{"ci"}
	created, err := apiClient.CreateTestSet(context.Background(), CreateTestSetInput{
		DisplayName:     "Provider test",
		Description:     &description,
		TestSetMetadata: &metadata,
		Tags:            &tags,
	})
	if err != nil {
		t.Fatalf("CreateTestSet(): %v", err)
	}
	if created.ID != "abc12345" || created.Slug != "provider-test" {
		t.Errorf("created = %#v", created)
	}
}

func TestGetAndUpdateTestSet(t *testing.T) {
	t.Parallel()

	requestCount := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount++
		if request.URL.Path != "/v1/test-sets/abc12345" {
			t.Fatalf("request path = %s", request.URL.Path)
		}
		switch requestCount {
		case 1:
			if request.Method != http.MethodGet {
				t.Fatalf("request method = %s, want GET", request.Method)
			}
			return testResponse(http.StatusOK, `{"test_set":{"name":"test-sets/abc12345","id":"abc12345","slug":"original","display_name":"Original","description":null,"test_set_type":null,"test_set_metadata":{},"parameters":{},"test_case_count":0,"tags":[],"create_time":"2026-09-14T00:00:00Z","update_time":null}}`, nil), nil
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
			if decoded["display_name"] != "Updated" || decoded["slug"] != "updated" {
				t.Errorf("request body = %#v", decoded)
			}
			return testResponse(http.StatusOK, `{"test_set":{"name":"test-sets/abc12345","id":"abc12345","slug":"updated","display_name":"Updated","description":"","test_set_type":"SCENARIO","test_set_metadata":{},"parameters":{},"test_case_count":0,"tags":[],"create_time":"2026-09-14T00:00:00Z","update_time":"2026-09-14T01:00:00Z"}}`, nil), nil
		default:
			t.Fatalf("unexpected request %d", requestCount)
			return nil, nil
		}
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if _, err := apiClient.GetTestSet(context.Background(), "abc12345"); err != nil {
		t.Fatalf("GetTestSet(): %v", err)
	}
	displayName := "Updated"
	slug := "updated"
	updated, err := apiClient.UpdateTestSet(context.Background(), "abc12345", UpdateTestSetInput{
		DisplayName: &displayName,
		Slug:        &slug,
	})
	if err != nil {
		t.Fatalf("UpdateTestSet(): %v", err)
	}
	if updated.DisplayName != "Updated" || updated.UpdateTime == nil {
		t.Errorf("updated = %#v", updated)
	}
}

func TestListTestSetsEncodesOptions(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/v1/test-sets" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("filter") != `display_name="Provider test"` || query.Get("page_size") != "100" || query.Get("page_token") != "next" || query.Get("order_by") != "-create_time" {
			t.Errorf("query = %v", query)
		}
		if got := query["tag_filters"]; !reflect.DeepEqual(got, []string{"ci", "terraform"}) {
			t.Errorf("tag_filters = %#v", got)
		}
		return testResponse(http.StatusOK, `{"test_sets":[],"next_page_token":""}`, nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	_, err = apiClient.ListTestSets(context.Background(), ListTestSetsOptions{
		Filter:     `display_name="Provider test"`,
		PageSize:   100,
		PageToken:  "next",
		OrderBy:    "-create_time",
		TagFilters: []string{"ci", "terraform"},
	})
	if err != nil {
		t.Fatalf("ListTestSets(): %v", err)
	}
}

func TestDeleteTestSet(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodDelete || request.URL.Path != "/v1/test-sets/abc12345" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		return testResponse(http.StatusOK, `{}`, nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := apiClient.DeleteTestSet(context.Background(), "abc12345"); err != nil {
		t.Fatalf("DeleteTestSet(): %v", err)
	}
}
