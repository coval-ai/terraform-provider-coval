package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// TestCase is the Coval public API representation of a test case.
type TestCase struct {
	Name               string          `json:"name"`
	ID                 string          `json:"id"`
	TestSetID          *string         `json:"test_set_id"`
	InputString        string          `json:"input_str"`
	ExpectedBehaviors  *[]string       `json:"expected_behaviors"`
	ExpectedOutputJSON json.RawMessage `json:"expected_output_json"`
	Description        *string         `json:"description"`
	InputType          *string         `json:"input_type"`
	ScriptTurns        json.RawMessage `json:"script_turns"`
	SimulationMetadata json.RawMessage `json:"simulation_metadata_input"`
	MetricInput        json.RawMessage `json:"metric_input"`
	UserNotes          *string         `json:"user_notes"`
	CreateTime         string          `json:"create_time"`
	UpdateTime         *string         `json:"update_time"`
}

// CreateTestCaseInput contains writable test-case fields.
type CreateTestCaseInput struct {
	TestSetID          string           `json:"test_set_id"`
	InputString        string           `json:"input_str"`
	ExpectedBehaviors  *[]string        `json:"expected_behaviors,omitempty"`
	ExpectedOutputJSON *json.RawMessage `json:"expected_output_json,omitempty"`
	Description        *string          `json:"description,omitempty"`
	InputType          *string          `json:"input_type,omitempty"`
	ScriptTurns        *json.RawMessage `json:"script_turns,omitempty"`
	SimulationMetadata *json.RawMessage `json:"simulation_metadata_input,omitempty"`
	MetricInput        *json.RawMessage `json:"metric_input,omitempty"`
	UserNotes          *string          `json:"user_notes,omitempty"`
}

// UpdateTestCaseInput contains test-case fields that may be changed in place.
type UpdateTestCaseInput struct {
	TestSetID          *string          `json:"test_set_id,omitempty"`
	InputString        *string          `json:"input_str,omitempty"`
	ExpectedBehaviors  *[]string        `json:"expected_behaviors,omitempty"`
	ExpectedOutputJSON *json.RawMessage `json:"expected_output_json,omitempty"`
	Description        *string          `json:"description,omitempty"`
	InputType          *string          `json:"input_type,omitempty"`
	ScriptTurns        *json.RawMessage `json:"script_turns,omitempty"`
	SimulationMetadata *json.RawMessage `json:"simulation_metadata_input,omitempty"`
	MetricInput        *json.RawMessage `json:"metric_input,omitempty"`
	UserNotes          *string          `json:"user_notes,omitempty"`
}

// ListTestCasesOptions filters and paginates test cases.
type ListTestCasesOptions struct {
	Filter    string
	PageSize  int
	PageToken string
	OrderBy   string
}

// ListTestCasesOutput is one page of test cases.
type ListTestCasesOutput struct {
	TestCases     []TestCase `json:"test_cases"`
	NextPageToken string     `json:"next_page_token"`
}

type testCaseEnvelope struct {
	TestCase TestCase `json:"test_case"`
}

// CreateTestCase creates a test case inside a test set.
func (c *Client) CreateTestCase(ctx context.Context, input CreateTestCaseInput) (TestCase, error) {
	var response testCaseEnvelope
	if err := c.Do(ctx, http.MethodPost, "test-cases", input, &response); err != nil {
		return TestCase{}, err
	}
	return response.TestCase, nil
}

// GetTestCase retrieves a test case by ID.
func (c *Client) GetTestCase(ctx context.Context, id string) (TestCase, error) {
	var response testCaseEnvelope
	if err := c.Do(ctx, http.MethodGet, "test-cases/"+url.PathEscape(id), nil, &response); err != nil {
		return TestCase{}, err
	}
	return response.TestCase, nil
}

// UpdateTestCase changes a test case in place.
func (c *Client) UpdateTestCase(ctx context.Context, id string, input UpdateTestCaseInput) (TestCase, error) {
	var response testCaseEnvelope
	if err := c.Do(ctx, http.MethodPatch, "test-cases/"+url.PathEscape(id), input, &response); err != nil {
		return TestCase{}, err
	}
	return response.TestCase, nil
}

// DeleteTestCase deletes a test case.
func (c *Client) DeleteTestCase(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "test-cases/"+url.PathEscape(id), nil, nil)
}

// ListTestCases returns one page of test cases.
func (c *Client) ListTestCases(ctx context.Context, options ListTestCasesOptions) (ListTestCasesOutput, error) {
	query := url.Values{}
	if options.Filter != "" {
		query.Set("filter", options.Filter)
	}
	if options.PageSize > 0 {
		query.Set("page_size", strconv.Itoa(options.PageSize))
	}
	if options.PageToken != "" {
		query.Set("page_token", options.PageToken)
	}
	if options.OrderBy != "" {
		query.Set("order_by", options.OrderBy)
	}

	requestPath := "test-cases"
	if encoded := query.Encode(); encoded != "" {
		requestPath += "?" + encoded
	}

	var response ListTestCasesOutput
	if err := c.Do(ctx, http.MethodGet, requestPath, nil, &response); err != nil {
		return ListTestCasesOutput{}, fmt.Errorf("list test cases: %w", err)
	}
	return response, nil
}
