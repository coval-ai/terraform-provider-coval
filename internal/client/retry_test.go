package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestDoRetriesOnlyRetryableStatuses(t *testing.T) {
	t.Parallel()

	for _, statusCode := range []int{408, 429, 500, 502, 503, 504} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()
			attempts := 0
			transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
				attempts++
				if attempts < defaultMaxAttempts {
					return testResponse(statusCode, `{}`, nil), nil
				}
				return testResponse(http.StatusOK, `{}`, nil), nil
			})
			client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
			if err != nil {
				t.Fatalf("New(): %v", err)
			}
			client.sleep = func(context.Context, time.Duration) error { return nil }
			if err := client.Do(context.Background(), http.MethodGet, "widgets", nil, nil); err != nil {
				t.Fatalf("Do(): %v", err)
			}
			if attempts != defaultMaxAttempts {
				t.Errorf("attempts = %d, want %d", attempts, defaultMaxAttempts)
			}
		})
	}
}

func TestDoDoesNotRetryOtherStatuses(t *testing.T) {
	t.Parallel()

	attempts := 0
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return testResponse(http.StatusNotImplemented, `{}`, nil), nil
	})
	client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := client.Do(context.Background(), http.MethodGet, "widgets", nil, nil); err == nil {
		t.Fatal("Do() succeeded on HTTP 501")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestDoDoesNotRetryMutations(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			attempts := 0
			transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
				attempts++
				return testResponse(http.StatusServiceUnavailable, `{}`, nil), nil
			})
			client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
			if err != nil {
				t.Fatalf("New(): %v", err)
			}
			if err := client.Do(context.Background(), method, "widgets", nil, nil); err == nil {
				t.Fatal("Do() succeeded on HTTP 503")
			}
			if attempts != 1 {
				t.Errorf("attempts = %d, want 1", attempts)
			}
		})
	}
}

func TestDoRetriesNetworkErrorsForSafeMethods(t *testing.T) {
	t.Parallel()

	attempts := 0
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		if attempts < defaultMaxAttempts {
			return nil, errors.New("temporary network failure")
		}
		return testResponse(http.StatusOK, `{}`, nil), nil
	})
	client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	client.sleep = func(context.Context, time.Duration) error { return nil }
	if err := client.Do(context.Background(), http.MethodGet, "widgets", nil, nil); err != nil {
		t.Fatalf("Do(): %v", err)
	}
	if attempts != defaultMaxAttempts {
		t.Errorf("attempts = %d, want %d", attempts, defaultMaxAttempts)
	}
}

func TestDoDoesNotRetryMutationNetworkErrors(t *testing.T) {
	t.Parallel()

	attempts := 0
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return nil, errors.New("network failure")
	})
	client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := client.Do(context.Background(), http.MethodPost, "widgets", nil, nil); err == nil {
		t.Fatal("Do() succeeded on network failure")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestBackoffUsesExponentialBoundedJitter(t *testing.T) {
	t.Parallel()

	client, err := New("secret", DefaultBaseURL)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	client.jitter = func() float64 { return 0 }
	if got := client.backoffDelay(1); got != 100*time.Millisecond {
		t.Errorf("first delay = %s, want 100ms", got)
	}
	if got := client.backoffDelay(2); got != 200*time.Millisecond {
		t.Errorf("second delay = %s, want 200ms", got)
	}

	client.jitter = func() float64 { return 1 }
	if got := client.backoffDelay(100); got != defaultMaxDelay {
		t.Errorf("capped delay = %s, want %s", got, defaultMaxDelay)
	}
}

func TestRetryAfter(t *testing.T) {
	t.Parallel()

	client, err := New("secret", DefaultBaseURL)
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	now := time.Date(2026, time.September, 14, 20, 0, 0, 0, time.UTC)
	client.now = func() time.Time { return now }
	client.jitter = func() float64 { return 0 }

	tests := map[string]struct {
		value string
		want  time.Duration
	}{
		"seconds":          {value: "2", want: 2 * time.Second},
		"decimal seconds":  {value: "0.25", want: 250 * time.Millisecond},
		"seconds capped":   {value: "20", want: defaultMaxDelay},
		"HTTP date":        {value: now.Add(3 * time.Second).Format(http.TimeFormat), want: 3 * time.Second},
		"past HTTP date":   {value: now.Add(-time.Second).Format(http.TimeFormat), want: 0},
		"invalid fallback": {value: "later", want: 100 * time.Millisecond},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := client.retryDelay(1, test.value); got != test.want {
				t.Errorf("retryDelay() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestDoUsesRetryAfter(t *testing.T) {
	t.Parallel()

	attempts := 0
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return testResponse(http.StatusTooManyRequests, `{}`, http.Header{"Retry-After": []string{"2"}}), nil
		}
		return testResponse(http.StatusOK, `{}`, nil), nil
	})
	client, err := New("secret", DefaultBaseURL, WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	var delays []time.Duration
	client.sleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}
	if err := client.Do(context.Background(), http.MethodGet, "widgets", nil, nil); err != nil {
		t.Fatalf("Do(): %v", err)
	}
	if len(delays) != 1 || delays[0] != 2*time.Second {
		t.Errorf("delays = %v, want [2s]", delays)
	}
}
