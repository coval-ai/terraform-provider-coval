package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Alert is the Coval public API representation of an alert definition.
type Alert struct {
	ID                    string          `json:"ulid"`
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	Status                string          `json:"status"`
	EvaluationType        string          `json:"evaluation_type"`
	ConversationSource    string          `json:"conversation_source"`
	MatchMode             string          `json:"match_mode"`
	CooldownSeconds       int64           `json:"cooldown_seconds"`
	CustomMessageTemplate *string         `json:"custom_message_template"`
	AgentIDs              []string        `json:"agent_ids"`
	RequiredTags          []string        `json:"required_tags"`
	ScheduledRunIDs       []string        `json:"scheduled_run_ids"`
	TriggerCount          int64           `json:"trigger_count"`
	LastTriggeredAt       *string         `json:"last_triggered_at"`
	Conditions            json.RawMessage `json:"conditions"`
	Channels              json.RawMessage `json:"channels"`
	CreateTime            string          `json:"create_time"`
	UpdateTime            string          `json:"update_time"`
}

// CreateAlertInput contains public writable alert fields.
type CreateAlertInput struct {
	Name                  string           `json:"name"`
	Description           *string          `json:"description,omitempty"`
	EvaluationType        string           `json:"evaluation_type"`
	ConversationSource    *string          `json:"conversation_source,omitempty"`
	MatchMode             *string          `json:"match_mode,omitempty"`
	CooldownSeconds       *int64           `json:"cooldown_seconds,omitempty"`
	CustomMessageTemplate *string          `json:"custom_message_template,omitempty"`
	AgentIDs              *[]string        `json:"agent_ids,omitempty"`
	RequiredTags          *[]string        `json:"required_tags,omitempty"`
	ScheduledRunIDs       *[]string        `json:"scheduled_run_ids,omitempty"`
	Conditions            json.RawMessage  `json:"conditions"`
	Channels              *json.RawMessage `json:"channels,omitempty"`
}

// UpdateAlertInput contains public writable alert fields sent to PATCH.
type UpdateAlertInput struct {
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	EvaluationType        string          `json:"evaluation_type"`
	ConversationSource    string          `json:"conversation_source"`
	MatchMode             string          `json:"match_mode"`
	CooldownSeconds       int64           `json:"cooldown_seconds"`
	CustomMessageTemplate *string         `json:"custom_message_template"`
	AgentIDs              []string        `json:"agent_ids"`
	RequiredTags          []string        `json:"required_tags"`
	ScheduledRunIDs       []string        `json:"scheduled_run_ids"`
	Conditions            json.RawMessage `json:"conditions"`
	Channels              json.RawMessage `json:"channels"`
}

// ListAlertsOptions filters and paginates alerts.
type ListAlertsOptions struct {
	ConversationSource string
	PageSize           int
	PageToken          string
}

// ListAlertsOutput is one page of alerts.
type ListAlertsOutput struct {
	Alerts        []Alert `json:"alerts"`
	NextPageToken *string `json:"next_page_token"`
	TotalCount    int64   `json:"total_count"`
}

// CreateAlert creates an alert.
func (c *Client) CreateAlert(ctx context.Context, input CreateAlertInput) (Alert, error) {
	var response Alert
	if err := c.Do(ctx, http.MethodPost, "alerts", input, &response); err != nil {
		return Alert{}, err
	}
	return response, nil
}

// GetAlert retrieves an alert by ID.
func (c *Client) GetAlert(ctx context.Context, id string) (Alert, error) {
	var response Alert
	if err := c.Do(ctx, http.MethodGet, "alerts/"+url.PathEscape(id), nil, &response); err != nil {
		return Alert{}, err
	}
	return response, nil
}

// UpdateAlert updates an alert.
func (c *Client) UpdateAlert(ctx context.Context, id string, input UpdateAlertInput) (Alert, error) {
	var response Alert
	if err := c.Do(ctx, http.MethodPatch, "alerts/"+url.PathEscape(id), input, &response); err != nil {
		return Alert{}, err
	}
	return response, nil
}

// DeleteAlert deletes an alert.
func (c *Client) DeleteAlert(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "alerts/"+url.PathEscape(id), nil, nil)
}

// ListAlerts returns one page of alerts.
func (c *Client) ListAlerts(ctx context.Context, options ListAlertsOptions) (ListAlertsOutput, error) {
	query := url.Values{}
	if options.ConversationSource != "" {
		query.Set("conversation_source", options.ConversationSource)
	}
	if options.PageSize > 0 {
		query.Set("page_size", strconv.Itoa(options.PageSize))
	}
	if options.PageToken != "" {
		query.Set("page_token", options.PageToken)
	}
	requestPath := "alerts"
	if encoded := query.Encode(); encoded != "" {
		requestPath += "?" + encoded
	}
	var response ListAlertsOutput
	if err := c.Do(ctx, http.MethodGet, requestPath, nil, &response); err != nil {
		return ListAlertsOutput{}, fmt.Errorf("list alerts: %w", err)
	}
	return response, nil
}
