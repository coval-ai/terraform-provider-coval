package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// RunTemplate is the Coval public API representation of a reusable run configuration.
type RunTemplate struct {
	Name            string          `json:"name"`
	ID              string          `json:"id"`
	DisplayName     string          `json:"display_name"`
	Description     string          `json:"description"`
	AgentIDs        []string        `json:"agent_ids"`
	PersonaIDs      []string        `json:"persona_ids"`
	TestSetIDs      []string        `json:"test_set_ids"`
	MetricIDs       []string        `json:"metric_ids"`
	MutationIDs     []string        `json:"mutation_ids"`
	IterationCount  int64           `json:"iteration_count"`
	Concurrency     int64           `json:"concurrency"`
	SubSampleSize   int64           `json:"sub_sample_size"`
	SubSampleSeed   *int64          `json:"sub_sample_seed"`
	Metadata        json.RawMessage `json:"metadata"`
	Tags            []string        `json:"tags"`
	CreateTime      string          `json:"create_time"`
	UpdateTime      *string         `json:"update_time"`
	CreatedByUserID *string         `json:"created_by_user_id"`
}

// CreateRunTemplateInput contains writable run-template fields.
type CreateRunTemplateInput struct {
	DisplayName    string           `json:"display_name"`
	Description    *string          `json:"description,omitempty"`
	AgentIDs       []string         `json:"agent_ids"`
	PersonaIDs     []string         `json:"persona_ids"`
	TestSetIDs     []string         `json:"test_set_ids"`
	MetricIDs      *[]string        `json:"metric_ids,omitempty"`
	MutationIDs    *[]string        `json:"mutation_ids,omitempty"`
	IterationCount int64            `json:"iteration_count"`
	Concurrency    int64            `json:"concurrency"`
	SubSampleSize  int64            `json:"sub_sample_size"`
	SubSampleSeed  *int64           `json:"sub_sample_seed,omitempty"`
	Metadata       *json.RawMessage `json:"metadata,omitempty"`
	Tags           *[]string        `json:"tags,omitempty"`
}

// UpdateRunTemplateInput contains writable run-template fields sent to PATCH.
type UpdateRunTemplateInput struct {
	DisplayName    string           `json:"display_name"`
	Description    string           `json:"description"`
	AgentIDs       []string         `json:"agent_ids"`
	PersonaIDs     []string         `json:"persona_ids"`
	TestSetIDs     []string         `json:"test_set_ids"`
	MetricIDs      []string         `json:"metric_ids"`
	MutationIDs    []string         `json:"mutation_ids"`
	IterationCount int64            `json:"iteration_count"`
	Concurrency    int64            `json:"concurrency"`
	SubSampleSize  int64            `json:"sub_sample_size"`
	SubSampleSeed  *int64           `json:"sub_sample_seed"`
	Metadata       *json.RawMessage `json:"metadata"`
	Tags           []string         `json:"tags"`
}

// ListRunTemplatesOptions filters and paginates run templates.
type ListRunTemplatesOptions struct {
	PageSize   int
	PageToken  string
	TagFilters []string
}

// ListRunTemplatesOutput is one page of run templates.
type ListRunTemplatesOutput struct {
	RunTemplates  []RunTemplate `json:"run_templates"`
	NextPageToken string        `json:"next_page_token"`
}

type runTemplateEnvelope struct {
	RunTemplate RunTemplate `json:"run_template"`
}

// CreateRunTemplate creates a run template.
func (c *Client) CreateRunTemplate(ctx context.Context, input CreateRunTemplateInput) (RunTemplate, error) {
	var response runTemplateEnvelope
	if err := c.Do(ctx, http.MethodPost, "run-templates", input, &response); err != nil {
		return RunTemplate{}, err
	}
	return response.RunTemplate, nil
}

// GetRunTemplate retrieves a run template by ID.
func (c *Client) GetRunTemplate(ctx context.Context, id string) (RunTemplate, error) {
	var response runTemplateEnvelope
	if err := c.Do(ctx, http.MethodGet, "run-templates/"+url.PathEscape(id), nil, &response); err != nil {
		return RunTemplate{}, err
	}
	return response.RunTemplate, nil
}

// UpdateRunTemplate changes a run template in place.
func (c *Client) UpdateRunTemplate(ctx context.Context, id string, input UpdateRunTemplateInput) (RunTemplate, error) {
	var response runTemplateEnvelope
	if err := c.Do(ctx, http.MethodPatch, "run-templates/"+url.PathEscape(id), input, &response); err != nil {
		return RunTemplate{}, err
	}
	return response.RunTemplate, nil
}

// DeleteRunTemplate deletes a run template.
func (c *Client) DeleteRunTemplate(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "run-templates/"+url.PathEscape(id), nil, nil)
}

// ListRunTemplates returns one page of run templates.
func (c *Client) ListRunTemplates(ctx context.Context, options ListRunTemplatesOptions) (ListRunTemplatesOutput, error) {
	query := url.Values{}
	if options.PageSize > 0 {
		query.Set("page_size", strconv.Itoa(options.PageSize))
	}
	if options.PageToken != "" {
		query.Set("page_token", options.PageToken)
	}
	for _, tag := range options.TagFilters {
		query.Add("tag_filters", tag)
	}

	requestPath := "run-templates"
	if encoded := query.Encode(); encoded != "" {
		requestPath += "?" + encoded
	}

	var response ListRunTemplatesOutput
	if err := c.Do(ctx, http.MethodGet, requestPath, nil, &response); err != nil {
		return ListRunTemplatesOutput{}, fmt.Errorf("list run templates: %w", err)
	}
	return response, nil
}
