package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	fiblog "github.com/gofiber/fiber/v2/middleware/logger"
)

// LogEntry represents a structured log entry in JSON.
type LogEntry struct {
	TimeStamp string `json:"timestamp"`
	Level     string `json:"level"`
	RequestID string `json:"request_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Message   string `json:"message"`
	Path      string `json:"path,omitempty"`
	Method    string `json:"method,omitempty"`
	Status    int    `json:"status,omitempty"`
	Latency   string `json:"latency,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Logger manages structured logging with rotation and color
type Logger struct {
	Mu         sync.Mutex
	Format     string
	TimeFormat string
	OutputDir  string
	MaxSizeMB  int
	File       *os.File
	FileSize   int64
	Log        *log.Logger
	FiberLog   fiber.Handler
}

// LoggerOption defines a function to configure the logger.
type LoggerOption func(*Logger)

func NewLogger(opts ...LoggerOption) (*Logger, error) {
	l := &Logger{
		Format:     "[${time}] ${status} - ${method} ${path} ${latency}\n",
		TimeFormat: time.RFC3339,
		OutputDir:  "./logs",
		MaxSizeMB:  10,
	}

	// Apply options to the logger
	for _, opt := range opts {
		opt(l)
	}

	if err := os.MkdirAll(l.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// Open initial Log file.
	file, err := OpenLogFile(l.OutputDir)
	if err != nil {
		return nil, err
	}

	l.File = file
	l.Log = log.New(file, "", 0)
	l.FiberLog = fiblog.New(fiblog.Config{
		Format:     l.Format,
		TimeFormat: l.TimeFormat,
		Output:     file,
	})

	return l, nil
}

// OpenLogFile opens a new log file with a timestamp of now.
func OpenLogFile(dir string) (*os.File, error) {
	filename := filepath.Join(dir, fmt.Sprintf(os.Getenv("APP")+"-%s.log", time.Now().Format("2006-12-02-15-04-05")))
	return os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

// Rotate checks file size and create new Log file if necessary.
func (l *Logger) Rotate() error {
	l.Mu.Lock()
	defer l.Mu.Unlock()

	info, err := l.File.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	// Checks and create new log file if exceed the file size. Default: 10MB
	if info.Size() >= int64(l.MaxSizeMB)*1024*1024 {
		l.File.Close()
		newFile, err := OpenLogFile(l.OutputDir)
		if err != nil {
			return err
		}
		l.File = newFile
		l.FileSize = 0
		l.Log.SetOutput(newFile)
		l.FiberLog = fiblog.New(fiblog.Config{
			Format:     l.Format,
			TimeFormat: l.TimeFormat,
			Output:     newFile,
		})
	}
	l.FileSize = info.Size()
	return nil
}

// WriteEntry writes a structured JSON log entry with color.
func (l *Logger) WriteEntry(entry LogEntry) error {
	if err := l.Rotate(); err != nil {
		return err
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %v", err)
	}

	l.Mu.Lock()
	defer l.Mu.Unlock()

	var colorPrefix, colorReset string
	switch entry.Level {
	case "INFO":
		colorPrefix = "\033[32m" // Green
	case "ERROR":
		colorPrefix = "\033[31m" // Red
	case "WARNING":
		colorPrefix = "" // Yellow
	default:
		colorPrefix = "\033[0m" // Red
	}
	colorReset = "\033[0m"

	// Write to file (JSON only, no color in file)
	l.Log.Output(2, string(data))

	// Write to terminal with color
	fmt.Fprintf(os.Stdout, "%s%s%s\n", colorPrefix, string(data), colorReset)

	return nil
}

// Middleware returns the Fiber logger middleware.
func (l *Logger) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := context.WithValue(c.Context(), "fiber_ctx", c)
		c.SetUserContext(ctx)
		return l.FiberLog(c)
	}
}

// SetupRoutesContext adds request ID and user ID to the context.
func SetupRoutesContext(c *fiber.Ctx) context.Context {
	ctx := c.UserContext()
	if ctx == nil {
		ctx = context.Background()
	}

	// feth request ID from request header.
	reqID := c.Get(fiber.HeaderXRequestID)
	if reqID == "" {
		reqID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	ctx = context.WithValue(ctx, "request_id", reqID)

	// fetch user ID from (set by JWT or locals)
	if userID, ok := c.Locals("user_id").(string); ok && userID != "" {
		ctx = context.WithValue(ctx, "user_id", userID)
	}

	return ctx
}

// SetupLogger initializes the logger and adds it to Fiber locals.
func SetupLogger(l *Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("logger", l)
		ctx := SetupRoutesContext(c)
		c.SetUserContext(ctx)
		return c.Next()
	}
}
