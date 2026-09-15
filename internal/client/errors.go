package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrorDetail describes a field-specific Coval API error.
type ErrorDetail struct {
	Field       string `json:"field,omitempty"`
	Description string `json:"description,omitempty"`
	HelpURL     string `json:"help_url,omitempty"`
}

// APIError is a non-successful response from the Coval API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    []ErrorDetail
	RequestID  string
}

func (e *APIError) Error() string {
	parts := []string{fmt.Sprintf("Coval API request failed with HTTP %d", e.StatusCode)}
	if e.Code != "" {
		parts = append(parts, "code "+e.Code)
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	if e.RequestID != "" {
		parts = append(parts, "request ID "+e.RequestID)
	}
	return strings.Join(parts, ": ")
}

// NetworkError is a transport failure before a Coval API response was received.
type NetworkError struct {
	Method   string
	Attempts int
	Cause    error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("Coval API %s request failed in transit after %d attempt(s)", e.Method, e.Attempts)
}

func (e *NetworkError) Unwrap() error {
	return e.Cause
}

// IsNotFound reports whether err is a Coval API not-found response.
func IsNotFound(err error) bool {
	var apiError *APIError
	return errors.As(err, &apiError) && apiError.StatusCode == http.StatusNotFound
}

type errorEnvelope struct {
	Error json.RawMessage `json:"error"`
}

type errorInformation struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details"`
}

func newAPIError(statusCode int, requestID string, body []byte) *APIError {
	apiError := &APIError{StatusCode: statusCode, RequestID: requestID}

	var envelope errorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil || len(envelope.Error) == 0 {
		return apiError
	}

	var information errorInformation
	if err := json.Unmarshal(envelope.Error, &information); err == nil {
		apiError.Code = information.Code
		apiError.Message = information.Message
		apiError.Details = information.Details
		return apiError
	}

	var message string
	if err := json.Unmarshal(envelope.Error, &message); err == nil {
		apiError.Message = message
	}
	return apiError
}
