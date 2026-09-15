package client

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDoReturnsStructuredAPIError(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		headers := http.Header{"X-Request-Id": []string{"request-123"}}
		return testResponse(http.StatusBadRequest, `{"error":{"code":"INVALID_ARGUMENT","message":"bad widget","details":[{"field":"name","description":"is required"}]},"private":"do-not-print"}`, headers), nil
	})
	client, err := New("secret-api-key", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	err = client.Do(context.Background(), http.MethodPost, "widgets", nil, nil)
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("Do() error = %T, want *APIError", err)
	}
	if apiError.StatusCode != http.StatusBadRequest || apiError.Code != "INVALID_ARGUMENT" || apiError.Message != "bad widget" || apiError.RequestID != "request-123" {
		t.Errorf("APIError = %#v", apiError)
	}
	if len(apiError.Details) != 1 || apiError.Details[0].Field != "name" {
		t.Errorf("Details = %#v", apiError.Details)
	}
	if strings.Contains(err.Error(), "do-not-print") || strings.Contains(err.Error(), "secret-api-key") {
		t.Errorf("error leaked protected content: %q", err)
	}
}

func TestDoHandlesAuthenticationErrorEnvelope(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testResponse(http.StatusUnauthorized, `{"error":"invalid API key","timestamp":"ignored"}`, nil), nil
	})
	client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	err = client.Do(context.Background(), http.MethodPost, "widgets", nil, nil)
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("Do() error = %T, want *APIError", err)
	}
	if apiError.Message != "invalid API key" || apiError.Code != "" {
		t.Errorf("APIError = %#v", apiError)
	}
}

func TestDoReturnsBoundedAPIErrorForUnknownEnvelope(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testResponse(http.StatusTeapot, "not-json-and-not-safe-to-echo", nil), nil
	})
	client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}

	err = client.Do(context.Background(), http.MethodPost, "widgets", nil, nil)
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("Do() error = %T, want *APIError", err)
	}
	if apiError.StatusCode != http.StatusTeapot {
		t.Errorf("StatusCode = %d", apiError.StatusCode)
	}
	if strings.Contains(err.Error(), "not-json") {
		t.Errorf("error echoed response payload: %q", err)
	}
}

func TestNetworkErrorDoesNotExposeCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("dial failed with secret-api-key")
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, cause
	})
	client, err := New("secret-api-key", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	client.sleep = func(context.Context, time.Duration) error { return nil }

	err = client.Do(context.Background(), http.MethodGet, "widgets", nil, nil)
	var networkError *NetworkError
	if !errors.As(err, &networkError) {
		t.Fatalf("Do() error = %T, want *NetworkError", err)
	}
	if networkError.Attempts != defaultMaxAttempts {
		t.Errorf("Attempts = %d, want %d", networkError.Attempts, defaultMaxAttempts)
	}
	if !errors.Is(err, cause) {
		t.Error("NetworkError does not unwrap to its cause")
	}
	if strings.Contains(err.Error(), "secret-api-key") {
		t.Errorf("error leaked API key: %q", err)
	}
}
