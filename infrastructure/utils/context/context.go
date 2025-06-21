package utils

import (
	"context"
	"time"
)

// ContextUtils provides utility functions for context management
type ContextUtils struct{}

// WithTimeout creates a context with timeout and returns the context and cancel function
func (cu *ContextUtils) WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

// WithCancel creates a cancellable context and returns the context and cancel function
func (cu *ContextUtils) WithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// WithDeadline creates a context with deadline and returns the context and cancel function
func (cu *ContextUtils) WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(parent, deadline)
}

// CheckContextError checks if context is cancelled or timeout and returns appropriate error
func (cu *ContextUtils) CheckContextError(ctx context.Context) error {
	return ctx.Err()
}

// IsCancelled checks if context is cancelled
func (cu *ContextUtils) IsCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// NewContextUtils creates a new instance of ContextUtils
func NewContextUtils() *ContextUtils {
	return &ContextUtils{}
}

// Global instance for convenience
var DefaultContextUtils = NewContextUtils()

// Convenience functions
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return DefaultContextUtils.WithTimeout(parent, timeout)
}

func WithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	return DefaultContextUtils.WithCancel(parent)
}

func WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return DefaultContextUtils.WithDeadline(parent, deadline)
}

func CheckContextError(ctx context.Context) error {
	return DefaultContextUtils.CheckContextError(ctx)
}

func IsCancelled(ctx context.Context) bool {
	return DefaultContextUtils.IsCancelled(ctx)
}

// WaitForContext waits for context to be done or timeout
func WaitForContext(ctx context.Context, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timeoutCtx.Done():
		return context.DeadlineExceeded
	}
}

// ContextWithValues creates a context with multiple values
func ContextWithValues(parent context.Context, keyValues map[string]interface{}) context.Context {
	ctx := parent
	for key, value := range keyValues {
		ctx = context.WithValue(ctx, key, value)
	}
	return ctx
}

// GetContextValue safely retrieves a value from context
func GetContextValue(ctx context.Context, key string) (interface{}, bool) {
	value := ctx.Value(key)
	return value, value != nil
}

// IsContextDone checks if context is done without blocking
func IsContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
