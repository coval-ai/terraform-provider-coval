package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func testResponse(statusCode int, body string, headers http.Header) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestNewValidatesAndNormalizesBaseURL(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		baseURL string
		want    string
		wantErr bool
	}{
		"production HTTPS":     {baseURL: "https://api.coval.dev/v1", want: "https://api.coval.dev/v1/"},
		"trailing slash":       {baseURL: "https://api.coval.dev/v1/", want: "https://api.coval.dev/v1/"},
		"localhost HTTP":       {baseURL: "http://localhost:8080/v1", want: "http://localhost:8080/v1/"},
		"IPv4 loopback HTTP":   {baseURL: "http://127.0.0.1:8080", want: "http://127.0.0.1:8080/"},
		"IPv6 loopback HTTP":   {baseURL: "http://[::1]:8080", want: "http://[::1]:8080/"},
		"empty":                {baseURL: "", wantErr: true},
		"relative":             {baseURL: "/v1", wantErr: true},
		"remote HTTP":          {baseURL: "http://api.coval.dev/v1", wantErr: true},
		"unsupported scheme":   {baseURL: "ftp://api.coval.dev/v1", wantErr: true},
		"embedded user info":   {baseURL: "https://user@example.com/v1", wantErr: true},
		"query on base URL":    {baseURL: "https://api.coval.dev/v1?debug=true", wantErr: true},
		"fragment on base URL": {baseURL: "https://api.coval.dev/v1#fragment", wantErr: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client, err := New("secret", test.baseURL)
			if test.wantErr {
				if err == nil {
					t.Fatalf("New() succeeded, base URL = %q", client.baseURL)
				}
				return
			}
			if err != nil {
				t.Fatalf("New(): %v", err)
			}
			if got := client.baseURL.String(); got != test.want {
				t.Errorf("base URL = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNewRejectsEmptyAPIKey(t *testing.T) {
	t.Parallel()
	if _, err := New("  ", DefaultBaseURL); err == nil {
		t.Fatal("New() succeeded with an empty API key")
	}
}

func TestDoBuildsAuthenticatedJSONRequest(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", request.Method)
		}
		if request.URL.String() != "https://example.com/v1/widgets?view=full" {
			t.Errorf("URL = %q", request.URL.String())
		}
		if got := request.Header["x-api-key"]; len(got) != 1 || got[0] != "secret" { //nolint:staticcheck // Coval's gateway requires this noncanonical casing.
			t.Errorf("lowercase x-api-key = %q, want secret", got)
		}
		if _, exists := request.Header["X-Api-Key"]; exists {
			t.Error("x-api-key was canonicalized")
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}

		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["name"] != "example" {
			t.Errorf("request name = %q", body["name"])
		}
		return testResponse(http.StatusOK, `{"widget":{"id":"abc"}}`, nil), nil
	})

	client, err := New(" secret ", "https://example.com/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	var response struct {
		Widget struct {
			ID string `json:"id"`
		} `json:"widget"`
	}
	if err := client.Do(context.Background(), http.MethodPost, "/widgets?view=full", map[string]string{"name": "example"}, &response); err != nil {
		t.Fatalf("Do(): %v", err)
	}
	if response.Widget.ID != "abc" {
		t.Errorf("response ID = %q, want abc", response.Widget.ID)
	}
}

func TestDoRejectsAbsoluteRequestURL(t *testing.T) {
	t.Parallel()
	client, err := New("secret", DefaultBaseURL)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := client.Do(context.Background(), http.MethodGet, "https://other.example/v1", nil, nil); err == nil {
		t.Fatal("Do() accepted an absolute request URL")
	}
}

func TestDoReportsEncodingAndDecodingErrors(t *testing.T) {
	t.Parallel()

	called := false
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return testResponse(http.StatusOK, "not-json", nil), nil
	})
	client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	if err := client.Do(context.Background(), http.MethodPost, "widgets", make(chan int), nil); err == nil {
		t.Fatal("Do() succeeded with an unencodable body")
	}
	if called {
		t.Fatal("transport was called for an unencodable body")
	}

	var destination map[string]any
	if err := client.Do(context.Background(), http.MethodGet, "widgets", nil, &destination); err == nil {
		t.Fatal("Do() succeeded with an invalid JSON response")
	}
}

func TestDoPropagatesContextCancellation(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Do(ctx, http.MethodGet, "widgets", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Do() error = %v, want context.Canceled", err)
	}
}

func TestForWorkspaceScopesOnlyTheDerivedClient(t *testing.T) {
	t.Parallel()

	headers := make(chan string, 2)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		headers <- request.Header.Get("X-Coval-Workspace-Id")
		return testResponse(http.StatusOK, `{}`, nil), nil
	})
	apiClient, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	if err := apiClient.ForWorkspace(" workspace-1 ").Do(context.Background(), http.MethodGet, "widgets", nil, nil); err != nil {
		t.Fatalf("scoped Do(): %v", err)
	}
	if err := apiClient.Do(context.Background(), http.MethodGet, "widgets", nil, nil); err != nil {
		t.Fatalf("base Do(): %v", err)
	}

	if got := <-headers; got != "workspace-1" {
		t.Errorf("scoped workspace header = %q, want workspace-1", got)
	}
	if got := <-headers; got != "" {
		t.Errorf("base workspace header = %q, want empty", got)
	}
}
