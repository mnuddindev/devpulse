package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

// WithMaxFileSize sets the maxiMum size of single Log file.
func WithMaxFileSize(size int) LoggerOption {
	return func(l *Logger) { l.MaxSizeMB = size }
}

// WithMaxDays sets the maxiMum age for the log files.
func WithMaxDays(days int) LoggerOption {
	return func(l *Logger) { l.MaxAgeDays = days }
}

// Debug logs a debug-level message with context and optional metadata.
func (l *Logger) Debug(ctx context.Context, msg string, meta map[string]string, fields ...interface{}) {
	l.LogWithLevel(ctx, LevelDebug, msg, meta, fields...)
}

// Info logs an info-level message with context and optional metadata.
func (l *Logger) Info(ctx context.Context, msg string, meta map[string]string, fields ...interface{}) {
	l.LogWithLevel(ctx, LevelInfo, msg, meta, fields...)
}

// Warn logs a warn-level message with context and optional metadata.
func (l *Logger) Warn(ctx context.Context, msg string, meta map[string]string, fields ...interface{}) {
	l.LogWithLevel(ctx, LevelWarn, msg, meta, fields...)
}

// Error logs an error-level message with context and optional metadata.
func (l *Logger) Error(ctx context.Context, msg string, meta map[string]string, fields ...interface{}) {
	l.LogWithLevel(ctx, LevelError, msg, meta, fields...)
}

// LogWithLevel logs a message with the specified level and context.
func (l *Logger) LogWithLevel(ctx context.Context, level LogLevel, msg string, meta map[string]string, fields ...interface{}) {
	entry := LogEntry{
		TimeStamp: time.Now().Format(l.TimeFormat),
		Level:     string(level),
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

	l.Queue <- entry
}

// Worker processes the async logging queue.
func (l *Logger) Worker() {
	for {
		select {
		case entry := <-l.Queue:
			l.WriteEntry(entry)
		case <-l.Quit:
			for len(l.Queue) > 0 {
				l.WriteEntry(<-l.Queue)
			}
			return
		}
	}
}

// CleanupOldLogs removes log files older than maxAgeDays.
func (l *Logger) CleanupOldLogs() {
	l.Mu.Lock()
	defer l.Mu.Unlock()

	files, err := filepath.Glob(filepath.Join(l.OutputDir, os.Getenv("APP")+"-*.log"))
	if err != nil {
		return
	}

	now := time.Now()
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()).Hours()/24 > float64(l.MaxAgeDays) {
			os.Remove(file)
		}
	}
}
