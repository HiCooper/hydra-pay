package middleware

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func StructuredLogger() gin.HandlerFunc {
	logger := log.New(os.Stdout, "", 0)
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		entry := map[string]interface{}{
			"timestamp":  start.Format(time.RFC3339),
			"level":      logLevel(c.Writer.Status()),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency_ms": time.Since(start).Milliseconds(),
			"client_ip":  c.ClientIP(),
		}
		b, _ := json.Marshal(entry)
		logger.Println(string(b))
	}
}

func logLevel(status int) string {
	if status >= 500 {
		return "error"
	}
	if status >= 400 {
		return "warn"
	}
	return "info"
}
