package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

type Logger struct {
	*slog.Logger
}

type LoggerConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`
	Format     string `yaml:"format" mapstructure:"format"` // "json" or "text"
	AddSource  bool   `yaml:"add_source" mapstructure:"add_source"`
	TimeFormat string `yaml:"time_format" mapstructure:"time_format"`
	Output     string `yaml:"output" mapstructure:"output"` // "stdout", "stderr", or file path
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
