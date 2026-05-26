package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hydra/pay-service/pkg/metrics"
)

// Metrics records HTTP request counts and latency for Prometheus scraping.
// Must be registered after RequestID and before StructuredLogger.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		metrics.HTTPRequestTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, path, status).Observe(duration)
	}
}
