package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hydra/pay-service/pkg/logger"
)

const ContextRequestID = logger.CtxRequestID

// RequestID reads or generates an X-Request-ID header, stores it in both
// gin context (for middleware access) and request context (for log correlation),
// and echoes it in the X-Request-ID response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set(ContextRequestID, id)
		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), id))
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
