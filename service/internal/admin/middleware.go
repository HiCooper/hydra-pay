package admin

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/hydra/pay-service/pkg/response"
)

// AdminAuth validates the X-Admin-Key header against the configured admin key.
func AdminAuth() gin.HandlerFunc {
	adminKey := os.Getenv("ADMIN_KEY")
	if adminKey == "" {
		adminKey = "admin-dev-key"
	}

	return func(c *gin.Context) {
		key := c.GetHeader("X-Admin-Key")
		if key == "" {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "X-Admin-Key header is required")
			c.Abort()
			return
		}
		if key != adminKey {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid admin key")
			c.Abort()
			return
		}
		c.Next()
	}
}
