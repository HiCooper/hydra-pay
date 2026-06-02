package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/hydra/pay-service/pkg/logger"
)

func StructuredRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				id, _ := c.Get(ContextRequestID)

				slog.LogAttrs(c.Request.Context(), slog.LevelError, "panic recovered",
					slog.Any("error", r),
					slog.String("stack", stack),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.String("client_ip", c.ClientIP()),
					slog.Any(logger.CtxRequestID, id),
				)

				// Store error detail so the access-log middleware also emits it.
				c.Set(logger.CtxErrorCode, "PANIC")
				c.Set(logger.CtxErrorMessage, "internal server error")

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}
