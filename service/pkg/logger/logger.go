package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// Init initializes the global structured logger with JSON output.
// Log level is controlled by the LOG_LEVEL environment variable (debug/info/warn/error), default info.
func Init() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

// WithRequestID stores a request ID in the context for later log extraction.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func extractAttrs(ctx context.Context, args []any) []any {
	if ctx == nil {
		return args
	}

	attrs := args

	if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
		attrs = append([]any{"request_id", id}, attrs...)
	}

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		attrs = append([]any{"trace_id", sc.TraceID().String()}, attrs...)
		attrs = append([]any{"span_id", sc.SpanID().String()}, attrs...)
	}

	return attrs
}

// Info logs at info level, extracting request_id from context if present.
func Info(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, extractAttrs(ctx, args)...)
}

// Warn logs at warn level, extracting request_id from context if present.
func Warn(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, extractAttrs(ctx, args)...)
}

// Error logs at error level, extracting request_id from context if present.
func Error(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, extractAttrs(ctx, args)...)
}

// Debug logs at debug level, extracting request_id from context if present.
func Debug(ctx context.Context, msg string, args ...any) {
	slog.DebugContext(ctx, msg, extractAttrs(ctx, args)...)
}
