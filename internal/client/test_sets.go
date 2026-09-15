package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// TestSet is the Coval public API representation of a test set.
type TestSet struct {
	Name            string          `json:"name"`
	ID              string          `json:"id"`
	Slug            string          `json:"slug"`
	DisplayName     string          `json:"display_name"`
	Description     *string         `json:"description"`
	TestSetType     *string         `json:"test_set_type"`
	TestSetMetadata json.RawMessage `json:"test_set_metadata"`
	Parameters      json.RawMessage `json:"parameters"`
	TestCaseCount   *int64          `json:"test_case_count"`
	Tags            []string        `json:"tags"`
	CreateTime      string          `json:"create_time"`
	UpdateTime      *string         `json:"update_time"`
}

// CreateTestSetInput contains writable test-set fields.
type CreateTestSetInput struct {
	DisplayName     string           `json:"display_name"`
	Slug            *string          `json:"slug,omitempty"`
	Description     *string          `json:"description,omitempty"`
	TestSetType     *string          `json:"test_set_type,omitempty"`
	TestSetMetadata *json.RawMessage `json:"test_set_metadata,omitempty"`
	Parameters      *json.RawMessage `json:"parameters,omitempty"`
	Tags            *[]string        `json:"tags,omitempty"`
}

// UpdateTestSetInput contains test-set fields that may be changed in place.
type UpdateTestSetInput struct {
	DisplayName     *string          `json:"display_name,omitempty"`
	Slug            *string          `json:"slug,omitempty"`
	Description     *string          `json:"description,omitempty"`
	TestSetType     *string          `json:"test_set_type,omitempty"`
	TestSetMetadata *json.RawMessage `json:"test_set_metadata,omitempty"`
	Parameters      *json.RawMessage `json:"parameters,omitempty"`
	Tags            *[]string        `json:"tags,omitempty"`
}

// ListTestSetsOptions filters and paginates test sets.
type ListTestSetsOptions struct {
	Filter     string
	PageSize   int
	PageToken  string
	OrderBy    string
	TagFilters []string
}

// ListTestSetsOutput is one page of test sets.
type ListTestSetsOutput struct {
	TestSets      []TestSet `json:"test_sets"`
	NextPageToken string    `json:"next_page_token"`
}

type testSetEnvelope struct {
	TestSet TestSet `json:"test_set"`
}

// CreateTestSet creates a test set.
func (c *Client) CreateTestSet(ctx context.Context, input CreateTestSetInput) (TestSet, error) {
	var response testSetEnvelope
	if err := c.Do(ctx, http.MethodPost, "test-sets", input, &response); err != nil {
		return TestSet{}, err
	}
	return response.TestSet, nil
}

// GetTestSet retrieves a test set by ID.
func (c *Client) GetTestSet(ctx context.Context, id string) (TestSet, error) {
	var response testSetEnvelope
	if err := c.Do(ctx, http.MethodGet, "test-sets/"+url.PathEscape(id), nil, &response); err != nil {
		return TestSet{}, err
	}
	return response.TestSet, nil
}

// UpdateTestSet changes a test set in place.
func (c *Client) UpdateTestSet(ctx context.Context, id string, input UpdateTestSetInput) (TestSet, error) {
	var response testSetEnvelope
	if err := c.Do(ctx, http.MethodPatch, "test-sets/"+url.PathEscape(id), input, &response); err != nil {
		return TestSet{}, err
	}
	return response.TestSet, nil
}

// DeleteTestSet deletes a test set.
func (c *Client) DeleteTestSet(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "test-sets/"+url.PathEscape(id), nil, nil)
}

// ListTestSets returns one page of test sets.
func (c *Client) ListTestSets(ctx context.Context, options ListTestSetsOptions) (ListTestSetsOutput, error) {
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
	for _, tag := range options.TagFilters {
		query.Add("tag_filters", tag)
	}

	requestPath := "test-sets"
	if encoded := query.Encode(); encoded != "" {
		requestPath += "?" + encoded
	}

	var response ListTestSetsOutput
	if err := c.Do(ctx, http.MethodGet, requestPath, nil, &response); err != nil {
		return ListTestSetsOutput{}, fmt.Errorf("list test sets: %w", err)
	}
	return response, nil
}
