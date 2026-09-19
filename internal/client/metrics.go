package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// MetricRuntimeConfig configures the model used for LLM-backed metrics.
type MetricRuntimeConfig struct {
	ModelVersion    *string `json:"model_version,omitempty"`
	ThinkingEnabled *bool   `json:"thinking_enabled,omitempty"`
}

// MetricTargetCondition defines which metric output counts as a success.
type MetricTargetCondition struct {
	ComparisonOperator string    `json:"comparison_operator"`
	TargetFloat        *float64  `json:"target_float,omitempty"`
	TargetValues       *[]string `json:"target_values,omitempty"`
}

// MetricEvaluation contains read-only evaluator catalog metadata.
type MetricEvaluation struct {
	Evaluator          string  `json:"evaluator"`
	OutputType         *string `json:"output_type"`
	OutputTypeSource   string  `json:"output_type_source"`
	SemanticType       *string `json:"semantic_type"`
	SemanticTypeSource string  `json:"semantic_type_source"`
}

// CurrentMetricVersion identifies the live metric version.
type CurrentMetricVersion struct {
	ULID          string `json:"ulid"`
	VersionNumber int64  `json:"version_number"`
	ChangeType    string `json:"change_type"`
}

// Metric is the Coval public API representation of a metric.
type Metric struct {
	Name                                string                 `json:"name"`
	ID                                  string                 `json:"id"`
	MetricName                          string                 `json:"metric_name"`
	Description                         string                 `json:"description"`
	MetricType                          string                 `json:"metric_type"`
	Evaluation                          *MetricEvaluation      `json:"evaluation"`
	Prompt                              *string                `json:"prompt"`
	EnabledTools                        *[]string              `json:"enabled_tools"`
	Categories                          *[]string              `json:"categories"`
	MinValue                            *float64               `json:"min_value"`
	MaxValue                            *float64               `json:"max_value"`
	MetadataFieldType                   *string                `json:"metadata_field_type"`
	MetadataFieldKey                    *string                `json:"metadata_field_key"`
	RegexPattern                        *string                `json:"regex_pattern"`
	Role                                *string                `json:"role"`
	MinPauseDurationSeconds             *float64               `json:"min_pause_duration_seconds"`
	MaxSilenceDurationSeconds           *float64               `json:"max_silence_duration_seconds"`
	MinSilenceGapSeconds                *float64               `json:"min_silence_gap_seconds"`
	FrequencyThreshold                  *float64               `json:"frequency_threshold"`
	Direction                           *string                `json:"direction"`
	SuccessSentiments                   *[]string              `json:"success_sentiments"`
	PercentAbove                        *float64               `json:"percent_above"`
	SuccessEndReasons                   *[]string              `json:"success_end_reasons"`
	ObservationName                     *string                `json:"observation_name"`
	ExpectedBody                        json.RawMessage        `json:"expected_body"`
	MatchPath                           *string                `json:"match_path"`
	MinVolumeChangeForPitchMisalignment *float64               `json:"min_volume_change_for_pitch_misalignment"`
	Threshold                           *int64                 `json:"threshold"`
	Operator                            *string                `json:"operator"`
	IVRFlow                             json.RawMessage        `json:"ivr_flow"`
	SQLQuery                            *string                `json:"sql_query"`
	AggregationMethod                   *string                `json:"aggregation_method"`
	Unit                                *string                `json:"unit"`
	CriteriaSource                      *string                `json:"criteria_source"`
	CriteriaPath                        *string                `json:"criteria_path"`
	Criteria                            *[]string              `json:"criteria"`
	ReportingMethod                     *string                `json:"reporting_method"`
	BasePromptTemplate                  *string                `json:"base_prompt_template"`
	IncludeTraces                       *bool                  `json:"include_traces"`
	RuntimeConfig                       *MetricRuntimeConfig   `json:"runtime_config"`
	TargetCondition                     *MetricTargetCondition `json:"target_condition"`
	Tags                                []string               `json:"tags"`
	CreatedBy                           *string                `json:"created_by"`
	CreateTime                          string                 `json:"create_time"`
	UpdateTime                          *string                `json:"update_time"`
	CurrentVersion                      *CurrentMetricVersion  `json:"current_version"`
}

// CreateMetricInput contains writable metric fields.
type CreateMetricInput struct {
	MetricName                          string                 `json:"metric_name"`
	Description                         string                 `json:"description"`
	MetricType                          string                 `json:"metric_type"`
	Prompt                              *string                `json:"prompt,omitempty"`
	EnabledTools                        *[]string              `json:"enabled_tools,omitempty"`
	Categories                          *[]string              `json:"categories,omitempty"`
	MinValue                            *float64               `json:"min_value,omitempty"`
	MaxValue                            *float64               `json:"max_value,omitempty"`
	MetadataFieldType                   *string                `json:"metadata_field_type,omitempty"`
	MetadataFieldKey                    *string                `json:"metadata_field_key,omitempty"`
	RegexPattern                        *string                `json:"regex_pattern,omitempty"`
	Role                                *string                `json:"role,omitempty"`
	MinPauseDurationSeconds             *float64               `json:"min_pause_duration_seconds,omitempty"`
	MaxSilenceDurationSeconds           *float64               `json:"max_silence_duration_seconds,omitempty"`
	MinSilenceGapSeconds                *float64               `json:"min_silence_gap_seconds,omitempty"`
	FrequencyThreshold                  *float64               `json:"frequency_threshold,omitempty"`
	Direction                           *string                `json:"direction,omitempty"`
	SuccessSentiments                   *[]string              `json:"success_sentiments,omitempty"`
	PercentAbove                        *float64               `json:"percent_above,omitempty"`
	SuccessEndReasons                   *[]string              `json:"success_end_reasons,omitempty"`
	ObservationName                     *string                `json:"observation_name,omitempty"`
	ExpectedBody                        *json.RawMessage       `json:"expected_body,omitempty"`
	MatchPath                           *string                `json:"match_path,omitempty"`
	MinVolumeChangeForPitchMisalignment *float64               `json:"min_volume_change_for_pitch_misalignment,omitempty"`
	Threshold                           *int64                 `json:"threshold,omitempty"`
	Operator                            *string                `json:"operator,omitempty"`
	IVRFlow                             *json.RawMessage       `json:"ivr_flow,omitempty"`
	SQLQuery                            *string                `json:"sql_query,omitempty"`
	AggregationMethod                   *string                `json:"aggregation_method,omitempty"`
	Unit                                *string                `json:"unit,omitempty"`
	CriteriaSource                      *string                `json:"criteria_source,omitempty"`
	CriteriaPath                        *string                `json:"criteria_path,omitempty"`
	Criteria                            *[]string              `json:"criteria,omitempty"`
	ReportingMethod                     *string                `json:"reporting_method,omitempty"`
	BasePromptTemplate                  *string                `json:"base_prompt_template,omitempty"`
	IncludeTraces                       *bool                  `json:"include_traces,omitempty"`
	RuntimeConfig                       *MetricRuntimeConfig   `json:"runtime_config,omitempty"`
	TargetCondition                     *MetricTargetCondition `json:"target_condition,omitempty"`
	Tags                                *[]string              `json:"tags,omitempty"`
}

// UpdateMetricInput contains writable metric fields for a partial update.
type UpdateMetricInput struct {
	CreateMetricInput
	ClearRuntimeConfig bool `json:"-"`
}

// MarshalJSON preserves the public API's distinction between an omitted
// runtime_config and an explicit null that restores the platform default.
func (input UpdateMetricInput) MarshalJSON() ([]byte, error) {
	encoded, err := json.Marshal(input.CreateMetricInput)
	if err != nil {
		return nil, err
	}
	if !input.ClearRuntimeConfig {
		return encoded, nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		return nil, err
	}
	object["runtime_config"] = json.RawMessage("null")
	return json.Marshal(object)
}

// ListMetricsOptions filters and paginates metrics.
type ListMetricsOptions struct {
	Filter         string
	PageSize       int
	PageToken      string
	OrderBy        string
	IncludeBuiltin bool
	TagFilters     []string
}

// ListMetricsOutput is one page of metrics.
type ListMetricsOutput struct {
	Metrics       []Metric `json:"metrics"`
	NextPageToken string   `json:"next_page_token"`
}

type metricEnvelope struct {
	Metric Metric `json:"metric"`
}

// CreateMetric creates a metric.
func (c *Client) CreateMetric(ctx context.Context, input CreateMetricInput) (Metric, error) {
	var response metricEnvelope
	if err := c.Do(ctx, http.MethodPost, "metrics", input, &response); err != nil {
		return Metric{}, err
	}
	return response.Metric, nil
}

// GetMetric retrieves a metric by ID.
func (c *Client) GetMetric(ctx context.Context, id string) (Metric, error) {
	var response metricEnvelope
	if err := c.Do(ctx, http.MethodGet, "metrics/"+url.PathEscape(id), nil, &response); err != nil {
		return Metric{}, err
	}
	return response.Metric, nil
}

// UpdateMetric changes a metric in place.
func (c *Client) UpdateMetric(ctx context.Context, id string, input UpdateMetricInput) (Metric, error) {
	var response metricEnvelope
	if err := c.Do(ctx, http.MethodPatch, "metrics/"+url.PathEscape(id), input, &response); err != nil {
		return Metric{}, err
	}
	return response.Metric, nil
}

// DeleteMetric deletes a metric.
func (c *Client) DeleteMetric(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "metrics/"+url.PathEscape(id), nil, nil)
}

// ListMetrics returns one page of metrics.
func (c *Client) ListMetrics(ctx context.Context, options ListMetricsOptions) (ListMetricsOutput, error) {
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
	if options.IncludeBuiltin {
		query.Set("include_builtin", "true")
	}
	for _, tag := range options.TagFilters {
		query.Add("tag_filters", tag)
	}

	requestPath := "metrics"
	if encoded := query.Encode(); encoded != "" {
		requestPath += "?" + encoded
	}
	var response ListMetricsOutput
	if err := c.Do(ctx, http.MethodGet, requestPath, nil, &response); err != nil {
		return ListMetricsOutput{}, fmt.Errorf("list metrics: %w", err)
	}
	return response, nil
}
