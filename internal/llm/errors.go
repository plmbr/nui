// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	stderrors "errors"
	"fmt"
)

// Sentinel errors for type checking with errors.Is().
var (
	ErrAuthentication = stderrors.New("authentication failed")
	ErrInvalidRequest = stderrors.New("invalid request")
	ErrModelNotFound  = stderrors.New("model not found")
	ErrProvider       = stderrors.New("provider error")
	ErrRateLimit      = stderrors.New("rate limit exceeded")
)

// APIError wraps provider HTTP errors with a sentinel for errors.Is.
type APIError struct {
	Provider string
	Status   int
	Code     string
	Err      error
	sentinel error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s", e.Provider, e.Err.Error())
	}
	return fmt.Sprintf("[%s] HTTP %d", e.Provider, e.Status)
}

func (e *APIError) Is(target error) bool {
	return e.sentinel != nil && target == e.sentinel
}

func (e *APIError) Unwrap() error {
	return e.Err
}

func newAPIError(provider string, status int, body string, sentinel error) *APIError {
	msg := body
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	return &APIError{
		Provider: provider,
		Status:   status,
		Err:      stderrors.New(msg),
		sentinel: sentinel,
	}
}

func classifyHTTPError(provider string, status int, body string) error {
	switch status {
	case 401:
		return newAPIError(provider, status, body, ErrAuthentication)
	case 404:
		return newAPIError(provider, status, body, ErrModelNotFound)
	case 429:
		return newAPIError(provider, status, body, ErrRateLimit)
	case 400:
		return newAPIError(provider, status, body, ErrInvalidRequest)
	default:
		if status >= 400 {
			return newAPIError(provider, status, body, ErrProvider)
		}
		return nil
	}
}
