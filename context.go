package tlog

import (
	"context"

	"go.uber.org/zap"
)

// Context keys for storing logger-related values.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"
	// UserIDKey is the context key for user ID.
	UserIDKey contextKey = "user_id"
	// TraceIDKey is the context key for trace ID.
	TraceIDKey contextKey = "trace_id"
)

// FromContext returns a logger with context fields (request_id, user_id, trace_id).
// If no context is provided or no fields are found, returns the global logger.
func FromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return L()
	}

	logger := L()
	var fields []zap.Field

	// Add request_id if present
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	// Add user_id if present
	if userID, ok := ctx.Value(UserIDKey).(uint); ok && userID > 0 {
		fields = append(fields, zap.Uint("user_id", userID))
	}

	// Add trace_id if present
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if len(fields) > 0 {
		return logger.With(fields...)
	}

	return logger
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// ContextWithFields adds multiple fields to the context at once.
func ContextWithFields(ctx context.Context, requestID string, userID uint, traceID string) context.Context {
	if requestID != "" {
		ctx = WithRequestID(ctx, requestID)
	}
	if userID > 0 {
		ctx = WithUserID(ctx, userID)
	}
	if traceID != "" {
		ctx = WithTraceID(ctx, traceID)
	}
	return ctx
}

// InfoCtx logs an info message with context fields.
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Info(msg, fields...)
}

// ErrorCtx logs an error message with context fields.
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Error(msg, fields...)
}

// DebugCtx logs a debug message with context fields.
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Debug(msg, fields...)
}

// WarnCtx logs a warning message with context fields.
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	FromContext(ctx).Warn(msg, fields...)
}
