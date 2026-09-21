// Package client provides authenticated access to the Coval public API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.coval.dev/v1"

	defaultMaxAttempts = 3
	defaultBaseDelay   = 200 * time.Millisecond
	defaultMaxDelay    = 5 * time.Second
	maxErrorBodyBytes  = 1 << 20
)

type sleepFunc func(context.Context, time.Duration) error

// Client is an authenticated Coval API client.
type Client struct {
	apiKey      string
	baseURL     *url.URL
	httpClient  *http.Client
	workspaceID string

	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	jitter      func() float64
	now         func() time.Time
	sleep       sleepFunc
}

// ForWorkspace returns an independent client that scopes requests to workspaceID.
func (c *Client) ForWorkspace(workspaceID string) *Client {
	derived := *c
	derived.workspaceID = strings.TrimSpace(workspaceID)
	return &derived
}

// Option customizes a Client.
type Option func(*Client)

// WithHTTPClient uses httpClient for requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

// New returns a client configured for the Coval public API.
func New(apiKey string, baseURL string, options ...Option) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("the Coval API key must not be empty")
	}

	parsedBaseURL, err := parseBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	client := &Client{
		apiKey:      apiKey,
		baseURL:     parsedBaseURL,
		httpClient:  http.DefaultClient,
		maxAttempts: defaultMaxAttempts,
		baseDelay:   defaultBaseDelay,
		maxDelay:    defaultMaxDelay,
		jitter:      rand.Float64,
		now:         time.Now,
		sleep:       sleepWithContext,
	}
	for _, option := range options {
		option(client)
	}

	return client, nil
}

// Do sends a JSON request and decodes a JSON response.
func (c *Client) Do(ctx context.Context, method string, requestPath string, requestBody any, responseBody any) error {
	endpoint, err := c.resolveURL(requestPath)
	if err != nil {
		return err
	}

	var encodedBody []byte
	if requestBody != nil {
		encodedBody, err = json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode Coval API request: %w", err)
		}
	}

	method = strings.ToUpper(method)
	attempts := 1
	if isRetryableMethod(method) {
		attempts = c.maxAttempts
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		request, requestErr := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(encodedBody))
		if requestErr != nil {
			return fmt.Errorf("create Coval API request: %w", requestErr)
		}

		// Preserve the lowercase spelling required by the public API contract.
		request.Header["x-api-key"] = []string{c.apiKey}
		if c.workspaceID != "" {
			request.Header.Set("X-Coval-Workspace-Id", c.workspaceID)
		}
		request.Header.Set("Accept", "application/json")
		if requestBody != nil {
			request.Header.Set("Content-Type", "application/json")
		}

		response, requestErr := c.httpClient.Do(request)
		if requestErr != nil {
			if response != nil {
				drainAndClose(response.Body)
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if attempt < attempts {
				if sleepErr := c.sleep(ctx, c.backoffDelay(attempt)); sleepErr != nil {
					return sleepErr
				}
				continue
			}
			return &NetworkError{Method: method, Attempts: attempt, Cause: requestErr}
		}

		if attempt < attempts && isRetryableStatus(response.StatusCode) {
			drainAndClose(response.Body)
			if sleepErr := c.sleep(ctx, c.retryDelay(attempt, response.Header.Get("Retry-After"))); sleepErr != nil {
				return sleepErr
			}
			continue
		}

		return decodeResponse(response, responseBody)
	}

	return &NetworkError{Method: method, Attempts: attempts, Cause: errors.New("retry loop exhausted")}
}

func parseBaseURL(rawBaseURL string) (*url.URL, error) {
	rawBaseURL = strings.TrimSpace(rawBaseURL)
	parsed, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse Coval API base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("the Coval API base URL must be an absolute URL")
	}
	if parsed.User != nil {
		return nil, errors.New("the Coval API base URL must not include user information")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("the Coval API base URL must not include a query or fragment")
	}

	switch parsed.Scheme {
	case "https":
	case "http":
		if !isLoopbackHost(parsed.Hostname()) {
			return nil, errors.New("the Coval API base URL must use HTTPS unless it addresses the local machine")
		}
	default:
		return nil, errors.New("the Coval API base URL must use HTTP or HTTPS")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/"
	return parsed, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) resolveURL(requestPath string) (string, error) {
	reference, err := url.Parse(strings.TrimLeft(requestPath, "/"))
	if err != nil {
		return "", fmt.Errorf("parse Coval API request path: %w", err)
	}
	if reference.IsAbs() || reference.Host != "" {
		return "", errors.New("the Coval API request path must be relative")
	}
	return c.baseURL.ResolveReference(reference).String(), nil
}

func decodeResponse(response *http.Response, destination any) error {
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, err := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		if err != nil {
			return fmt.Errorf("read Coval API error response: %w", err)
		}
		return newAPIError(response.StatusCode, response.Header.Get("X-Request-Id"), body)
	}
	if destination == nil || response.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("decode Coval API response: %w", err)
	}
	return nil
}

func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 32<<10))
	_ = body.Close()
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
