package client

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func isRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func (c *Client) retryDelay(attempt int, retryAfter string) time.Duration {
	if delay, ok := c.parseRetryAfter(retryAfter); ok {
		return delay
	}
	return c.backoffDelay(attempt)
}

func (c *Client) parseRetryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil && seconds >= 0 {
		return minDuration(time.Duration(seconds*float64(time.Second)), c.maxDelay), true
	}
	when, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	return minDuration(maxDuration(when.Sub(c.now()), 0), c.maxDelay), true
}

func (c *Client) backoffDelay(attempt int) time.Duration {
	delay := c.baseDelay
	for currentAttempt := 1; currentAttempt < attempt && delay < c.maxDelay; currentAttempt++ {
		if delay > c.maxDelay/2 {
			delay = c.maxDelay
			break
		}
		delay *= 2
	}
	delay = minDuration(delay, c.maxDelay)
	return time.Duration(float64(delay) * (0.5 + c.jitter()*0.5))
}

func minDuration(left time.Duration, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}

func maxDuration(left time.Duration, right time.Duration) time.Duration {
	if left > right {
		return left
	}
	return right
}
