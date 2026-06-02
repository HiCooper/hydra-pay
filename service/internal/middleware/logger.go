package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydra/pay-service/pkg/logger"
)

// StructuredLogger is an enterprise-grade access-log middleware.
//
// Every request emits a single JSON log line with:
//   - timestamp, level, message (standard slog fields)
//   - request_id, trace_id, span_id  (correlation — from context)
//   - method, path, query, status, latency_ms, response_size, client_ip, user_agent (HTTP semantics)
//   - error_code, error_message (only on 4xx/5xx — extracted from gin context set by response.Error)
//
// Log levels map to HTTP status: 5xx → ERROR, 4xx → WARN, 2xx/3xx → INFO.
// Health/metrics endpoints are sampled at DEBUG to reduce noise (configurable via LOG_LEVEL).
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		// Sample health/metrics probes at DEBUG to keep production logs clean.
		path := c.Request.URL.Path
		if (path == "/health" || path == "/metrics") && status < 400 {
			level = slog.LevelDebug
		}

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", c.Request.URL.RawQuery),
			slog.Int("status", status),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.Int("response_size", c.Writer.Size()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		}

		// Attach request_id from gin context (set by RequestID middleware).
		if id, ok := c.Get(logger.CtxRequestID); ok {
			attrs = append(attrs, slog.Any(logger.CtxRequestID, id))
		}

		// Attach error details for non-success responses (set by response.Error).
		if status >= 400 {
			if code, ok := c.Get(logger.CtxErrorCode); ok {
				attrs = append(attrs, slog.String("error_code", code.(string)))
			}
			if msg, ok := c.Get(logger.CtxErrorMessage); ok {
				attrs = append(attrs, slog.String("error", msg.(string)))
			}
		}

		slog.LogAttrs(c.Request.Context(), level, "http request completed", attrs...)
	}
}
