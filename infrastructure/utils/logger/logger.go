package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	zap    *zap.Logger
	fields map[string]interface{}
}

type LoggerConfig struct {
	Level       string `yaml:"Level" mapstructure:"Level"`
	Format      string `yaml:"Format" mapstructure:"Format"` // "json" or "compact"
	AddSource   bool   `yaml:"AddSource" mapstructure:"AddSource"`
	TimeFormat  string `yaml:"TimeFormat" mapstructure:"TimeFormat"`
	Output      string `yaml:"Output" mapstructure:"Output"`           // "stdout", "stderr", or file path (used when EnableFile is false)
	EnableColor bool   `yaml:"EnableColor" mapstructure:"EnableColor"` // Enable colored output
	EnableFile  bool   `yaml:"EnableFile" mapstructure:"EnableFile"`   // Enable file logging (writes to both console and file)
	FilePath    string `yaml:"FilePath" mapstructure:"FilePath"`       // File path when EnableFile is true
}

var (
	defaultLogger *Logger
	contextKey    = &struct{ name string }{"logger"}
)

// Initialize sets up the global logger with the provided configuration
func Initialize(config LoggerConfig) error {
	level, err := parseZapLogLevel(config.Level)
	if err != nil {
		return err
	}

	// Create output writers
	var writers []zapcore.WriteSyncer

	// Always add console output
	var consoleOutput zapcore.WriteSyncer
	switch config.Output {
	case "", "stdout":
		consoleOutput = zapcore.AddSync(os.Stdout)
	case "stderr":
		consoleOutput = zapcore.AddSync(os.Stderr)
	default:
		// If output is a file path but EnableFile is false, treat as console output
		if !config.EnableFile {
			file, err := getOutputFile(config.Output)
			if err != nil {
				return fmt.Errorf("failed to initialize output: %w", err)
			}
			consoleOutput = zapcore.AddSync(file)
		} else {
			consoleOutput = zapcore.AddSync(os.Stdout)
		}
	}
	writers = append(writers, consoleOutput)

	// Add file output if enabled (in addition to console)
	if config.EnableFile && config.FilePath != "" {
		file, err := getOutputFile(config.FilePath)
		if err != nil {
			return fmt.Errorf("failed to initialize file output: %w", err)
		}
		writers = append(writers, zapcore.AddSync(file))
	}

	// Combine all writers
	var output zapcore.WriteSyncer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = zapcore.NewMultiWriteSyncer(writers...)
	}

	// Create encoder config with proper caller information
	var encoderConfig zapcore.EncoderConfig
	if config.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "time"
		encoderConfig.LevelKey = "level"
		encoderConfig.MessageKey = "msg"
		encoderConfig.CallerKey = "source"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		// Compact format
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.TimeKey = "time"
		encoderConfig.LevelKey = "level"
		encoderConfig.MessageKey = "msg"
		encoderConfig.CallerKey = "source"
		encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02T15:04:05.000-0700"))
		}
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

		if config.EnableColor {
			encoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
		}
	}

	// Create encoder
	var encoder zapcore.Encoder
	if config.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = NewZapCompactEncoder(encoderConfig, config.EnableColor)
	}

	// Create core with proper caller skip
	core := zapcore.NewCore(encoder, output, level)

	var zapLogger *zap.Logger
	if config.AddSource {
		// Add caller with proper skip level to get real caller
		zapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	} else {
		zapLogger = zap.New(core)
	}

	defaultLogger = &Logger{
		zap:    zapLogger,
		fields: make(map[string]interface{}),
	}

	return nil
}

// GetDefault returns the default logger instance
func GetDefault() *Logger {
	if defaultLogger == nil {
		// Fallback to a basic logger if not initialized
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		zapLogger, _ := config.Build(zap.AddCallerSkip(1))
		defaultLogger = &Logger{
			zap:    zapLogger,
			fields: make(map[string]interface{}),
		}
	}
	return defaultLogger
}

// WithFields creates a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return &Logger{
		zap:    l.zap.With(zapFields...),
		fields: newFields,
	}
}

// WithField creates a new logger with a single additional field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.WithFields(map[string]interface{}{key: value})
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

// Core logging methods
func (l *Logger) Info(msg string, args ...any) {
	if len(args) > 0 {
		zapFields := make([]zap.Field, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				zapFields = append(zapFields, zap.Any(key, args[i+1]))
			}
		}
		l.zap.Info(msg, zapFields...)
	} else {
		l.zap.Info(msg)
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	if len(args) > 0 {
		zapFields := make([]zap.Field, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				zapFields = append(zapFields, zap.Any(key, args[i+1]))
			}
		}
		l.zap.Debug(msg, zapFields...)
	} else {
		l.zap.Debug(msg)
	}
}

func (l *Logger) Warn(msg string, args ...any) {
	if len(args) > 0 {
		zapFields := make([]zap.Field, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				zapFields = append(zapFields, zap.Any(key, args[i+1]))
			}
		}
		l.zap.Warn(msg, zapFields...)
	} else {
		l.zap.Warn(msg)
	}
}

func (l *Logger) Error(msg string, args ...any) {
	if len(args) > 0 {
		zapFields := make([]zap.Field, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				zapFields = append(zapFields, zap.Any(key, args[i+1]))
			}
		}
		l.zap.Error(msg, zapFields...)
	} else {
		l.zap.Error(msg)
	}
}

// Formatted logging methods for Logger struct
// Infof logs at Info level with printf-style formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.zap.Info(msg)
}

// Debugf logs at Debug level with printf-style formatting
func (l *Logger) Debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.zap.Debug(msg)
}

// Warnf logs at Warn level with printf-style formatting
func (l *Logger) Warnf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.zap.Warn(msg)
}

// Errorf logs at Error level with printf-style formatting
func (l *Logger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.zap.Error(msg)
}

// Fatalf logs at Error level with printf-style formatting and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.zap.Fatal(msg)
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

// Convenience functions for default logger with proper caller information
func Info(msg string, args ...any) {
	GetDefault().zap.Info(msg, convertToZapFields(args...)...)
}

func Debug(msg string, args ...any) {
	GetDefault().zap.Debug(msg, convertToZapFields(args...)...)
}

func Warn(msg string, args ...any) {
	GetDefault().zap.Warn(msg, convertToZapFields(args...)...)
}

func Error(msg string, args ...any) {
	GetDefault().zap.Error(msg, convertToZapFields(args...)...)
}

func Fatal(msg string, args ...any) {
	GetDefault().zap.Fatal(msg, convertToZapFields(args...)...)
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
func parseZapLogLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug", "DEBUG":
		return zap.DebugLevel, nil
	case "info", "INFO":
		return zap.InfoLevel, nil
	case "warn", "WARN", "warning", "WARNING":
		return zap.WarnLevel, nil
	case "error", "ERROR":
		return zap.ErrorLevel, nil
	default:
		return zap.InfoLevel, nil
	}
}

func getOutputFile(output string) (*os.File, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
	}

	file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", output, err)
	}
	return file, nil
}

func convertToZapFields(args ...any) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	zapFields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			zapFields = append(zapFields, zap.Any(key, args[i+1]))
		}
	}
	return zapFields
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

// NewZapCompactEncoder creates a custom Zap encoder for compact format
func NewZapCompactEncoder(cfg zapcore.EncoderConfig, enableColor bool) zapcore.Encoder {
	if enableColor {
		cfg.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	} else {
		cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	}
	return zapcore.NewConsoleEncoder(cfg)
}
