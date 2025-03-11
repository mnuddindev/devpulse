// Package utils provides utility functions for the BlogBlaze application.
package utils

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// Common error types for reuse.
var (
	ErrBadRequest          = NewError(fiber.StatusBadRequest, "Invalid request")
	ErrUnauthorized        = NewError(fiber.StatusUnauthorized, "Unauthorized")
	ErrForbidden           = NewError(fiber.StatusForbidden, "Forbidden")
	ErrNotFound            = NewError(fiber.StatusNotFound, "Resource not found")
	ErrInternalServerError = NewError(fiber.StatusInternalServerError, "Internal server error")
)

// Error represents a structured error for the web app.
type CustomError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewError creates a new Error with a status code, message, and optional details.
func NewError(code int, message string, details ...string) *CustomError {
	e := &CustomError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		e.Details = details[0]
	}
	return e
}

// Error implements the error interface.
func (e *CustomError) Error() string {
	return fmt.Sprintf("status %d: %s", e.Code, e.Message)
}

// WithCause attaches underlying details to the error (renamed from your intent).
func (e *CustomError) WithCause(err error) *CustomError {
	if err != nil {
		e.Details = err.Error()
	}
	return e
}

// HandleError sends a standardized error response using GoFiber.
func HandleError(c *fiber.Ctx, err error) error {
	var appErr *CustomError

	if As(err, &appErr) {
		if appErr.Code >= 500 {
			appErr.Details = ""
		}
		return c.Status(appErr.Code).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    appErr.Code,
				"message": appErr.Message,
				"details": appErr.Details,
			},
		})
	}

	// Fallback for unhandled errors
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    fiber.StatusInternalServerError,
			"message": "Something went wrong",
		},
	})
}

// WrapError wraps an existing error with a custom status and message.
func WrapError(err error, code int, message string) *CustomError {
	return NewError(code, message, err.Error())
}

// As is a helper to unwrap errors (replacing errors.As for clarity in this package).
func As(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*CustomError); ok {
		if t, ok := target.(**CustomError); ok {
			*t = e
			return true
		}
	}
	return false
}
