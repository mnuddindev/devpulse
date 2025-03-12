package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LogBuilder builds a log entry with a fluent interface.
type LogBuilder struct {
	Logger *Logger
	Ctx    context.Context
	Level  LogLevel
	Msg    string
	Meta   map[string]string
	Fields []interface{}
}

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

// Debug starts a debug-level log entry.
func (l *Logger) Debug(ctx context.Context) *LogBuilder {
	return &LogBuilder{Logger: l, Ctx: ctx, Level: LevelDebug}
}

// Info starts an info-level log entry.
func (l *Logger) Info(ctx context.Context) *LogBuilder {
	return &LogBuilder{Logger: l, Ctx: ctx, Level: LevelInfo}
}

// Warn starts a warn-level log entry.
func (l *Logger) Warn(ctx context.Context) *LogBuilder {
	return &LogBuilder{Logger: l, Ctx: ctx, Level: LevelWarn}
}

// Error starts an error-level log entry.
func (l *Logger) Error(ctx context.Context) *LogBuilder {
	return &LogBuilder{Logger: l, Ctx: ctx, Level: LevelError}
}

// WithMeta adds metadata to the log entry.
func (b *LogBuilder) WithMeta(meta map[string]string) *LogBuilder {
	b.Meta = meta
	return b
}

// WithFields adds formatted fields to the message.
func (b *LogBuilder) WithFields(fields ...interface{}) *LogBuilder {
	b.Fields = fields
	return b
}

// LogWithLevel logs a message with the specified level and context.
func (b *LogBuilder) Logs(msg string) {
	entry := LogEntry{
		TimeStamp: time.Now().Format(b.Logger.TimeFormat),
		Level:     string(b.Level),
		Message:   fmt.Sprintf(msg, b.Fields...),
	}

	// Extract request context
	if reqID, ok := b.Ctx.Value("request_id").(string); ok {
		entry.RequestID = reqID
	}
	if userID, ok := b.Ctx.Value("user_id").(string); ok {
		entry.UserID = userID
	}

	// Add Fiber-specific fields if available
	if c, ok := b.Ctx.Value("fiber_ctx").(*fiber.Ctx); ok {
		entry.Path = c.Path()
		entry.Method = c.Method()
		entry.Status = c.Response().StatusCode()
		entry.Latency = time.Since(c.Context().Time()).String()
	}

	b.Logger.Queue <- entry
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
func (l *Logger) CleanupOldLogs(ctx context.Context) error {
	l.Mu.Lock()
	defer l.Mu.Unlock()

	files, err := filepath.Glob(filepath.Join(l.OutputDir, os.Getenv("APP")+"-*.log"))
	if err != nil {
		return nil
	}

	now := time.Now()
	for _, file := range files {
		select {
		case <-ctx.Done():
			return fmt.Errorf("log cleanup canceled: %w", ctx.Err())
		default:
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			if now.Sub(info.ModTime()).Hours()/24 > float64(l.MaxAgeDays) {
				if err := os.Remove(file); err != nil {
					return fmt.Errorf("failed to remove old log file %s: %w", file, err)
				}
			}
		}
	}
	return nil
}
