package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// WithFormat sets the Fiber logger format.
func WithFormat(format string) LoggerOption {
	return func(l *Logger) { l.Format = format }
}

// WithTimeFormat sets the timestamp format.
func WithTimeFormat(timeformat string) LoggerOption {
	return func(l *Logger) { l.TimeFormat = timeformat }
}

// WithOutputDir sets the output directory of Log File.
func WithOutputDir(dir string) LoggerOption {
	return func(l *Logger) { l.OutputDir = dir }
}

// WithMaxFileSize sets the maximum size of single Log file.
func WithMaxFileSize(size int) LoggerOption {
	return func(l *Logger) { l.MaxSizeMB = size }
}

// Info logs an info-level message with context.
func (l *Logger) Info(ctx context.Context, msg string, fields ...interface{}) {
	l.LogWithLevel(ctx, "INFO", msg, fields...)
}

// Error logs an error-level message with context.
func (l *Logger) Error(ctx context.Context, msg string, fields ...interface{}) {
	l.LogWithLevel(ctx, "ERROR", msg, fields...)
}

// Error logs an error-level message with context.
func (l *Logger) Warning(ctx context.Context, msg string, fields ...interface{}) {
	l.LogWithLevel(ctx, "WARNING", msg, fields...)
}

// LogWithLevel logs a message with the specified level and context.
func (l *Logger) LogWithLevel(ctx context.Context, level, msg string, fields ...interface{}) {
	entry := LogEntry{
		TimeStamp: time.Now().Format(l.TimeFormat),
		Level:     level,
		Message:   fmt.Sprintf(msg, fields...),
	}

	// Extract request context
	if reqID, ok := ctx.Value("request_id").(string); ok {
		entry.RequestID = reqID
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		entry.UserID = userID
	}

	// Add Fiber-specific fields if available
	if c, ok := ctx.Value("fiber_ctx").(*fiber.Ctx); ok {
		entry.Path = c.Path()
		entry.Method = c.Method()
		entry.Status = c.Response().StatusCode()
		entry.Latency = time.Since(c.Context().Time()).String()
	}

	l.WriteEntry(entry)
}
