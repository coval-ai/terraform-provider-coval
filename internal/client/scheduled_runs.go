package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ScheduledRun is the Coval public API representation of a recurring run schedule.
type ScheduledRun struct {
	Name               string  `json:"name"`
	ID                 string  `json:"id"`
	DisplayName        string  `json:"display_name"`
	RunTemplateID      string  `json:"run_template_id"`
	ScheduleExpression string  `json:"schedule_expression"`
	ScheduleTimezone   string  `json:"schedule_timezone"`
	Enabled            bool    `json:"enabled"`
	LastRunAt          *string `json:"last_run_at"`
	LastRunID          *string `json:"last_run_id"`
	CreateTime         string  `json:"create_time"`
	UpdateTime         *string `json:"update_time"`
}

// CreateScheduledRunInput contains writable scheduled-run fields.
type CreateScheduledRunInput struct {
	DisplayName        string `json:"display_name"`
	RunTemplateID      string `json:"run_template_id"`
	ScheduleExpression string `json:"schedule_expression"`
	ScheduleTimezone   string `json:"schedule_timezone"`
	Enabled            bool   `json:"enabled"`
}

// UpdateScheduledRunInput contains writable scheduled-run fields sent to PATCH.
type UpdateScheduledRunInput = CreateScheduledRunInput

// ListScheduledRunsOptions filters and paginates scheduled runs.
type ListScheduledRunsOptions struct {
	PageSize      int
	PageToken     string
	Enabled       *bool
	RunTemplateID string
}

// ListScheduledRunsOutput is one page of scheduled runs.
type ListScheduledRunsOutput struct {
	ScheduledRuns []ScheduledRun `json:"scheduled_runs"`
	NextPageToken *string        `json:"next_page_token"`
	TotalCount    int64          `json:"total_count"`
}

type scheduledRunEnvelope struct {
	ScheduledRun ScheduledRun `json:"scheduled_run"`
}

// CreateScheduledRun creates a scheduled run.
func (c *Client) CreateScheduledRun(ctx context.Context, input CreateScheduledRunInput) (ScheduledRun, error) {
	var response scheduledRunEnvelope
	if err := c.Do(ctx, http.MethodPost, "scheduled-runs", input, &response); err != nil {
		return ScheduledRun{}, err
	}
	return response.ScheduledRun, nil
}

// GetScheduledRun retrieves a scheduled run by ID.
func (c *Client) GetScheduledRun(ctx context.Context, id string) (ScheduledRun, error) {
	var response scheduledRunEnvelope
	if err := c.Do(ctx, http.MethodGet, "scheduled-runs/"+url.PathEscape(id), nil, &response); err != nil {
		return ScheduledRun{}, err
	}
	return response.ScheduledRun, nil
}

// UpdateScheduledRun updates a scheduled run.
func (c *Client) UpdateScheduledRun(ctx context.Context, id string, input UpdateScheduledRunInput) (ScheduledRun, error) {
	var response scheduledRunEnvelope
	if err := c.Do(ctx, http.MethodPatch, "scheduled-runs/"+url.PathEscape(id), input, &response); err != nil {
		return ScheduledRun{}, err
	}
	return response.ScheduledRun, nil
}

// DeleteScheduledRun deletes a scheduled run.
func (c *Client) DeleteScheduledRun(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "scheduled-runs/"+url.PathEscape(id), nil, nil)
}

// ListScheduledRuns returns one page of scheduled runs.
func (c *Client) ListScheduledRuns(ctx context.Context, options ListScheduledRunsOptions) (ListScheduledRunsOutput, error) {
	query := url.Values{}
	if options.PageSize > 0 {
		query.Set("page_size", strconv.Itoa(options.PageSize))
	}
	if options.PageToken != "" {
		query.Set("page_token", options.PageToken)
	}
	if options.Enabled != nil {
		query.Set("enabled", strconv.FormatBool(*options.Enabled))
	}
	if options.RunTemplateID != "" {
		query.Set("template_id", options.RunTemplateID)
	}
	requestPath := "scheduled-runs"
	if encoded := query.Encode(); encoded != "" {
		requestPath += "?" + encoded
	}
	var response ListScheduledRunsOutput
	if err := c.Do(ctx, http.MethodGet, requestPath, nil, &response); err != nil {
		return ListScheduledRunsOutput{}, fmt.Errorf("list scheduled runs: %w", err)
	}
	return response, nil
}
