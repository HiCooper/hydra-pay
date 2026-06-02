package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// Context keys shared across middleware and handlers.
const (
	CtxRequestID  = "request_id"
	CtxErrorCode  = "log_error_code"
	CtxErrorMessage = "log_error_message"
)

// Init initializes the global structured JSON logger.
//
// LOG_LEVEL controls verbosity: debug | info | warn | error (default: info).
// Set LOG_ADD_SOURCE=true to emit file:line in every log entry (useful in dev, off in prod for perf).
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

	addSource := strings.ToLower(os.Getenv("LOG_ADD_SOURCE")) == "true"

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
		// Replace attr keys to match common enterprise conventions.
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "timestamp"
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				a.Key = "level"
			case slog.SourceKey:
				a.Key = "caller"
			}
			return a
		},
	})
	slog.SetDefault(slog.New(handler))
}

// extractAttrs prepends request_id, trace_id, and span_id from context when available.
func extractAttrs(ctx context.Context, args []any) []any {
	if ctx == nil {
		return args
	}

	attrs := args

	if id, ok := ctx.Value(CtxRequestID).(string); ok && id != "" {
		attrs = append([]any{CtxRequestID, id}, attrs...)
	}

	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		attrs = append([]any{"trace_id", sc.TraceID().String()}, attrs...)
		attrs = append([]any{"span_id", sc.SpanID().String()}, attrs...)
	}

	return attrs
}

// WithRequestID stores a request ID in the context for log correlation.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxRequestID, id)
}

// Info logs at info level with context-derived attributes.
func Info(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, extractAttrs(ctx, args)...)
}

// Warn logs at warn level with context-derived attributes.
func Warn(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, extractAttrs(ctx, args)...)
}

// Error logs at error level with context-derived attributes.
func Error(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, extractAttrs(ctx, args)...)
}

// Debug logs at debug level with context-derived attributes.
func Debug(ctx context.Context, msg string, args ...any) {
	slog.DebugContext(ctx, msg, extractAttrs(ctx, args)...)
}
