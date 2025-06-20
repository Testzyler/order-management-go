package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Logger struct {
	*slog.Logger
}

type LoggerConfig struct {
	Level       string `yaml:"Level" mapstructure:"level"`
	Format      string `yaml:"Format" mapstructure:"format"` // "json" or "compact"
	AddSource   bool   `yaml:"AddSource" mapstructure:"add_source"`
	TimeFormat  string `yaml:"TimeFormat" mapstructure:"time_format"`
	Output      string `yaml:"Output" mapstructure:"output"`             // "stdout", "stderr", or file path
	EnableColor bool   `yaml:"EnableColor" mapstructure:"enable_color"` // Enable colored output
}

var (
	defaultLogger *Logger
	contextKey    = &struct{ name string }{"logger"}
)

// Initialize sets up the global logger with the provided configuration
func Initialize(config LoggerConfig) error {
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return err
	}

	output, err := getOutput(config.Output)
	if err != nil {
		return err
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: config.AddSource,
	}

	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "compact":
		handler = NewCompactTextHandler(output, opts, config.EnableColor)
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	slogLogger := slog.New(handler)
	defaultLogger = &Logger{Logger: slogLogger}

	// Set as default slog logger
	slog.SetDefault(slogLogger)

	return nil
}

// GetDefault returns the default logger instance
func GetDefault() *Logger {
	if defaultLogger == nil {
		// Fallback to a basic logger if not initialized
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
		defaultLogger = &Logger{Logger: slog.New(handler)}
	}
	return defaultLogger
}

// WithFields creates a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for key, value := range fields {
		args = append(args, key, value)
	}
	return &Logger{Logger: l.Logger.With(args...)}
}

// WithField creates a new logger with a single additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{Logger: l.Logger.With(key, value)}
}

// WithRequestID creates a new logger with a request ID field
func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.WithField("request_id", requestID)
}

// WithUserID creates a new logger with a user ID field
func (l *Logger) WithUserID(userID string) *Logger {
	return l.WithField("user_id", userID)
}

// WithComponent creates a new logger with a component field
func (l *Logger) WithComponent(component string) *Logger {
	return l.WithField("component", component)
}

// WithError creates a new logger with an error field
func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err.Error())
}

// Formatted logging methods for Logger struct
// Infof logs at Info level with printf-style formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}

// Debugf logs at Debug level with printf-style formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, args...))
}

// Warnf logs at Warn level with printf-style formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs at Error level with printf-style formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}

// Fatalf logs at Error level with printf-style formatting and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Context operations
// ToContext adds the logger to the context
func (l *Logger) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey, l)
}

// FromContext retrieves the logger from context, fallback to default if not found
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(contextKey).(*Logger); ok {
		return logger
	}
	return GetDefault()
}

// Request ID context operations
var requestIDKey = &struct{ name string }{"request_id"}

// WithRequestIDToContext adds a request ID to the context
func WithRequestIDToContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext retrieves the request ID from context
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// LoggerWithRequestIDFromContext creates a logger with request ID from context
func LoggerWithRequestIDFromContext(ctx context.Context) *Logger {
	requestID := RequestIDFromContext(ctx)
	if requestID != "" {
		return GetDefault().WithRequestID(requestID)
	}
	return GetDefault()
}

// Convenience functions for default logger
func Info(msg string, args ...any) {
	GetDefault().Info(msg, args...)
}

func Debug(msg string, args ...any) {
	GetDefault().Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	GetDefault().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	GetDefault().Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	GetDefault().Error(msg, args...)
	os.Exit(1)
}

// Formatted convenience functions for default logger
func Infof(format string, args ...interface{}) {
	GetDefault().Infof(format, args...)
}

func Debugf(format string, args ...interface{}) {
	GetDefault().Debugf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	GetDefault().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	GetDefault().Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	GetDefault().Fatalf(format, args...)
}

// WithFields creates a new logger with additional fields from default logger
func WithFields(fields map[string]interface{}) *Logger {
	return GetDefault().WithFields(fields)
}

// WithField creates a new logger with a single additional field from default logger
func WithField(key string, value interface{}) *Logger {
	return GetDefault().WithField(key, value)
}

// WithRequestID creates a new logger with a request ID field from default logger
func WithRequestID(requestID string) *Logger {
	return GetDefault().WithRequestID(requestID)
}

// WithComponent creates a new logger with a component field from default logger
func WithComponent(component string) *Logger {
	return GetDefault().WithComponent(component)
}

// Helper functions
func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug, nil
	case "info", "INFO":
		return slog.LevelInfo, nil
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn, nil
	case "error", "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, nil
	}
}

func getOutput(output string) (io.Writer, error) {
	switch output {
	case "", "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		// Assume it's a file path
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		return file, nil
	}
}

// Structured logging helpers
func LogHTTPRequest(logger *Logger, method, path string, statusCode int, duration time.Duration, requestID string) {
	logger.WithFields(map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"request_id":  requestID,
		"type":        "http_request",
	}).Info("HTTP request processed")
}

func LogDatabaseQuery(logger *Logger, query string, duration time.Duration, requestID string) {
	logger.WithFields(map[string]interface{}{
		"query":       query,
		"duration_ms": duration.Milliseconds(),
		"request_id":  requestID,
		"type":        "database_query",
	}).Debug("Database query executed")
}

func LogServiceCall(logger *Logger, service, method string, duration time.Duration, requestID string) {
	logger.WithFields(map[string]interface{}{
		"service":     service,
		"method":      method,
		"duration_ms": duration.Milliseconds(),
		"request_id":  requestID,
		"type":        "service_call",
	}).Debug("Service method called")
}

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorGray   = "\033[90m" // Timestamp
	ColorBlue   = "\033[34m" // Debug
	ColorGreen  = "\033[32m" // Info
	ColorYellow = "\033[33m" // Warn
	ColorRed    = "\033[31m" // Error
	ColorCyan   = "\033[36m" // Source
)

// CompactTextHandler is a custom slog handler that formats logs in a compact format
// Format: 2025-06-20T21:26:54.635+0700    info    cmd/cmd.go:52   message
type CompactTextHandler struct {
	writer      io.Writer
	opts        *slog.HandlerOptions
	enableColor bool
}

func NewCompactTextHandler(w io.Writer, opts *slog.HandlerOptions, enableColor bool) *CompactTextHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	// For now, trust the user's enableColor setting
	// Terminal detection can be unreliable in some environments
	actualEnableColor := enableColor

	return &CompactTextHandler{
		writer:      w,
		opts:        opts,
		enableColor: actualEnableColor,
	}
}

func (h *CompactTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *CompactTextHandler) Handle(ctx context.Context, record slog.Record) error {
	var buf strings.Builder

	// Timestamp with timezone (gray if colors enabled)
	if h.enableColor {
		buf.WriteString(ColorGray)
	}
	buf.WriteString(record.Time.Format("2006-01-02T15:04:05.000-0700"))
	if h.enableColor {
		buf.WriteString(ColorReset)
	}
	buf.WriteString("\t")

	// Level (lowercase with color)
	levelStr := strings.ToLower(record.Level.String())
	if h.enableColor {
		switch record.Level {
		case slog.LevelDebug:
			buf.WriteString(ColorBlue)
		case slog.LevelInfo:
			buf.WriteString(ColorGreen)
		case slog.LevelWarn:
			buf.WriteString(ColorYellow)
		case slog.LevelError:
			buf.WriteString(ColorRed)
		}
	}
	buf.WriteString(levelStr)
	if h.enableColor {
		buf.WriteString(ColorReset)
	}
	buf.WriteString("\t")

	// Source file and line (cyan if colors enabled)
	if h.opts.AddSource && record.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := fs.Next()
		if f.File != "" {
			// Extract just the filename, not the full path
			filename := filepath.Base(f.File)
			buf.WriteString(fmt.Sprintf("%s:%d", filename, f.Line))
			if h.enableColor {
				buf.WriteString(ColorReset)
			}
		}
	}
	buf.WriteString("\t")

	// Message (no color)
	buf.WriteString(record.Message)

	// Add attributes if any
	record.Attrs(func(attr slog.Attr) bool {
		buf.WriteString(" ")
		buf.WriteString(attr.Key)
		if h.enableColor {
			buf.WriteString(ColorReset)
		}
		buf.WriteString("=")
		buf.WriteString(fmt.Sprintf("%v", attr.Value.Any()))
		return true
	})

	buf.WriteString("\n")

	_, err := h.writer.Write([]byte(buf.String()))
	return err
}

func (h *CompactTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, return the same handler
	// In a full implementation, you'd create a new handler with additional attrs
	return h
}

func (h *CompactTextHandler) WithGroup(name string) slog.Handler {
	// For simplicity, return the same handler
	// In a full implementation, you'd handle grouping
	return h
}
