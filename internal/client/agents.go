package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Agent is the Coval public API representation of an agent.
type Agent struct {
	ID               string          `json:"id"`
	CustomerAgentID  string          `json:"customer_agent_id"`
	DisplayName      string          `json:"display_name"`
	ModelType        string          `json:"model_type"`
	PhoneNumber      *string         `json:"phone_number"`
	Endpoint         *string         `json:"endpoint"`
	Prompt           *string         `json:"prompt"`
	Language         *string         `json:"language"`
	Attributes       json.RawMessage `json:"attributes"`
	Metadata         json.RawMessage `json:"metadata"`
	Workflows        json.RawMessage `json:"workflows"`
	MetricIDs        []string        `json:"metric_ids"`
	TestSetIDs       []string        `json:"test_set_ids"`
	KnowledgeBaseIDs []string        `json:"knowledge_base_ids"`
	Tags             []string        `json:"tags"`
	CreateTime       string          `json:"create_time"`
	UpdateTime       *string         `json:"update_time"`
}

// CreateAgentInput contains writable agent fields.
type CreateAgentInput struct {
	DisplayName     string           `json:"display_name"`
	ModelType       string           `json:"model_type"`
	PhoneNumber     *string          `json:"phone_number,omitempty"`
	Endpoint        *string          `json:"endpoint,omitempty"`
	Prompt          *string          `json:"prompt,omitempty"`
	CustomerAgentID *string          `json:"customer_agent_id,omitempty"`
	Language        *string          `json:"language,omitempty"`
	Attributes      *json.RawMessage `json:"attributes,omitempty"`
	Metadata        *json.RawMessage `json:"metadata,omitempty"`
	Workflows       *json.RawMessage `json:"workflows,omitempty"`
	MetricIDs       *[]string        `json:"metric_ids,omitempty"`
	TestSetIDs      *[]string        `json:"test_set_ids,omitempty"`
	Tags            *[]string        `json:"tags,omitempty"`
}

// UpdateAgentInput contains the complete writable agent state sent on update.
type UpdateAgentInput struct {
	DisplayName     string          `json:"display_name"`
	ModelType       string          `json:"model_type"`
	PhoneNumber     *string         `json:"phone_number"`
	Endpoint        *string         `json:"endpoint"`
	Prompt          *string         `json:"prompt"`
	CustomerAgentID string          `json:"customer_agent_id"`
	Language        *string         `json:"language"`
	Attributes      json.RawMessage `json:"attributes"`
	Metadata        json.RawMessage `json:"metadata"`
	Workflows       json.RawMessage `json:"workflows"`
	MetricIDs       []string        `json:"metric_ids"`
	TestSetIDs      []string        `json:"test_set_ids"`
	Tags            []string        `json:"tags"`
}

// ListAgentsOptions filters and paginates agents.
type ListAgentsOptions struct {
	Filter     string
	PageSize   int
	PageToken  string
	OrderBy    string
	TagFilters []string
}

// ListAgentsOutput is one page of agents.
type ListAgentsOutput struct {
	Agents        []Agent `json:"agents"`
	NextPageToken string  `json:"next_page_token"`
}

type agentEnvelope struct {
	Agent Agent `json:"agent"`
}

// CreateAgent creates an agent.
func (c *Client) CreateAgent(ctx context.Context, input CreateAgentInput) (Agent, error) {
	var response agentEnvelope
	if err := c.Do(ctx, http.MethodPost, "agents", input, &response); err != nil {
		return Agent{}, err
	}
	return response.Agent, nil
}

// GetAgent retrieves an agent by ID.
func (c *Client) GetAgent(ctx context.Context, id string) (Agent, error) {
	var response agentEnvelope
	if err := c.Do(ctx, http.MethodGet, "agents/"+url.PathEscape(id), nil, &response); err != nil {
		return Agent{}, err
	}
	return response.Agent, nil
}

// UpdateAgent changes an agent in place.
func (c *Client) UpdateAgent(ctx context.Context, id string, input UpdateAgentInput) (Agent, error) {
	var response agentEnvelope
	if err := c.Do(ctx, http.MethodPatch, "agents/"+url.PathEscape(id), input, &response); err != nil {
		return Agent{}, err
	}
	return response.Agent, nil
}

// DeleteAgent deletes an agent.
func (c *Client) DeleteAgent(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "agents/"+url.PathEscape(id), nil, nil)
}

// ListAgents returns one page of agents.
func (c *Client) ListAgents(ctx context.Context, options ListAgentsOptions) (ListAgentsOutput, error) {
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

	requestPath := "agents"
	if encoded := query.Encode(); encoded != "" {
		requestPath += "?" + encoded
	}

	var response ListAgentsOutput
	if err := c.Do(ctx, http.MethodGet, requestPath, nil, &response); err != nil {
		return ListAgentsOutput{}, fmt.Errorf("list agents: %w", err)
	}
	return response, nil
}
