package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit wraps the request body with MaxBytesReader to enforce a maximum
// request body size. Requests exceeding the limit will cause body parsing
// (ShouldBindJSON etc.) to fail with an error.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
