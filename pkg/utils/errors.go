package utils

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// CustomError represents a structured error for the web page
type CustomError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// NewError creates a new CustomError with a status code and message
func NewError(code int, message string, details ...string) *CustomError {
	ce := &CustomError{
		Status:  code,
		Message: message,
	}
	if len(details) > 0 {
		ce.Details = details[0]
	}
	return ce
}

// Error implements the error interface
func (ce *CustomError) Error() string {
	return fmt.Sprintf("status %d: %s", ce.Status, ce.Message)
}

// Common error types for reuse
var (
	ErrBadRequest     = NewError(fiber.StatusBadRequest, "Invalid request")
	ErrUnauthorized   = NewError(fiber.StatusUnauthorized, "Unauthorized")
	ErrForbidden      = NewError(fiber.StatusForbidden, "Forbidden")
	ErrNotFound       = NewError(fiber.StatusNotFound, "Resource not found")
	ErrInternalServer = NewError(fiber.StatusInternalServerError, "Internal server error")
)

// HandleError sends a standardized error response using GoFiber
func HandleError(c *fiber.Ctx, err error) error {
	var CustomErr *CustomError

	// Check is the error is already a CustomError
	if errors.As(err, &CustomErr) {
		if CustomErr.Status >= 500 {
			CustomErr.Details = ""
		}
		return c.Status(CustomErr.Status).JSON(fiber.Map{
			"error": fiber.Map{
				"status":  CustomErr.Status,
				"message": CustomErr.Message,
				"details": CustomErr.Details,
			},
		})
	}

	// Fallback call for unhandled errors
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": fiber.Map{
			"status":  fiber.StatusInternalServerError,
			"message": "Something went wrong",
		},
	})
}

// WrapError wraps an existing error with a custom status and message
func WrapError(err error, code int, message string) *CustomError {
	return NewError(code, message, err.Error())
}
