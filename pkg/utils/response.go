package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mnuddindev/devpulse/pkg/logger"
)

// Response holds a standardized API response fields.
type Response struct {
	Success bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   *CustomError
}

// ResponseBuilder builds a response with a fluent interface.
type ResponseBuilder struct {
	Ctx     context.Context
	C       *fiber.Ctx
	Success bool
	Message string
	Data    interface{}
	Err     *CustomError
}

// Success sends a standardized success response.
func Success(c *fiber.Ctx) *ResponseBuilder {
	return &ResponseBuilder{
		Ctx:     c.UserContext(),
		C:       c,
		Success: true,
	}
}

// Error sends a standardized error response using CustomError.
func Error(c *fiber.Ctx, err *CustomError) *ResponseBuilder {
	return &ResponseBuilder{
		Ctx:     c.UserContext(),
		C:       c,
		Success: false,
		Err:     err,
	}
}

// WithMessage adds a custom message to the response.
func (b *ResponseBuilder) WithMessage(msg string) *ResponseBuilder {
	b.Message = msg
	return b
}

// WithData adds data to the response.
func (b *ResponseBuilder) WithData(data interface{}) *ResponseBuilder {
	b.Data = data
	return b
}

// Send sends the response and logs it.
func (b *ResponseBuilder) Send() error {
	resp := Response{
		Success: b.Success,
		Message: b.Message,
		Data:    b.Data,
		Error:   b.Err,
	}

	status := fiber.StatusOK
	if !b.Success && b.Err != nil {
		status = b.Err.Code
	}

	if logger, ok := b.C.Locals("logger").(*logger.Logger); ok {
		meta := map[string]string{
			"status":  fmt.Sprintf("%d", status),
			"success": fmt.Sprintf("%t", b.Success),
			"path":    b.C.Path(),
			"method":  b.C.Method(),
			"latency": time.Since(b.C.Context().Time()).String(),
		}
		if b.Success {
			logger.Info(b.Ctx).WithMeta(meta).Logs("Response sent")
		} else {
			logger.Error(b.Ctx).WithMeta(meta).Logs(fmt.Sprintf("Error response sent: %s", b.Err.Error()))
		}
	}

	return b.C.Status(status).JSON(resp)
}

// SendError is a convenience function to send an error response directly.
func SendError(c *fiber.Ctx, err error) error {
	appErr, ok := err.(*CustomError)
	if !ok {
		appErr = ErrInternalServerError.WithCause(err)
	}
	return Error(c, appErr).Send()
}

// SendSuccess is a convenience function to send a success response directly.
func SendSuccess(c *fiber.Ctx, data interface{}) error {
	return Success(c).WithData(data).Send()
}
