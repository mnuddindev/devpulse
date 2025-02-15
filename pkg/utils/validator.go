package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrResponse represents the structure of the error response.
type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

// Error represents a single validation error.
type Error struct {
	Field string `json:"field"`
	Msg   string `json:"msg"`
}

// Validator is a struct that holds the validator instance from the go-playground/validator package
type Validator struct {
	validator *validator.Validate
}

// NewValidator is a function that returns a new instance of the Validator struct
func NewValidator() *Validator {
	v := validator.New()

	// Register custom validation functions here
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{validator: v}
}

// Validate is a method that validates the input struct and returns a map of errors
// The map is formatted as JSON-friendly output for client-side consumption.
func (v *Validator) Validate(str interface{}) *ErrorResponse {
	err := v.validator.Struct(str)
	if err == nil {
		return nil
	}
	response := ErrorResponse{Errors: make([]Error, 0, len(err.(validator.ValidationErrors)))}
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, err := range validationErrors {
			field := err.Field()                                // get the field that caused the error
			tag := err.Tag()                                    // get the tag that caused the error
			message := getErrorMessage(field, tag, err.Param()) // get the error message
			// append the error to the response
			response.Errors = append(response.Errors, Error{Field: field, Msg: message})
		}
	}
	return &response
}

// getErrorMessage is a helper function that returns the error message based on the field and tag
func getErrorMessage(field, tag, param string) string {
	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, param)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of the following values: %s", field, param)
	case "eqfiled":
		return fmt.Sprintf("%s must be equal to %s", field, param)
	default:
		return fmt.Sprintf("something wrong on %s; %s", field, tag)
	}
}
