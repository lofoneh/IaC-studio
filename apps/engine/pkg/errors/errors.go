package errors

import (
	"errors"
	"fmt"
)

// Code represents a stable error code for programmatic handling.
type Code string

const (
	CodeUnknown       Code = "unknown"
	CodeInvalid       Code = "invalid"
	CodeNotFound      Code = "not_found"
	CodeConflict      Code = "conflict"
	CodeUnauthorized  Code = "unauthorized"
	CodeForbidden     Code = "forbidden"
	CodeInternal      Code = "internal"
	CodeUnavailable   Code = "unavailable"
	CodeDeadline      Code = "deadline_exceeded"
	CodeAlreadyExists Code = "already_exists"
)

// AppError is a structured error type that carries a code, message, and optional metadata.
type AppError struct {
	Code    Code
	Message string
	Err     error
	Meta    map[string]any
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error for errors.Is/As support.
func (e *AppError) Unwrap() error { return e.Err }

// WithMeta attaches metadata to the error.
func (e *AppError) WithMeta(k string, v any) *AppError {
	if e.Meta == nil {
		e.Meta = map[string]any{}
	}
	e.Meta[k] = v
	return e
}

// New creates a new AppError with code and message.
func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap wraps an existing error with code and message.
func Wrap(err error, code Code, message string) *AppError {
	if err == nil {
		return New(code, message)
	}
	return &AppError{Code: code, Message: message, Err: err}
}

// IsCode checks if an error has the provided code (through unwrapping).
func IsCode(err error, code Code) bool {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code == code
	}
	return false
}
