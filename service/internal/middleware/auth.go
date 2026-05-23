package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/response"
)

const ContextAppID = "app_id"

// App is a minimal struct for API key validation.
type App struct {
	ID     uuid.UUID `gorm:"column:id"`
	APIKey string    `gorm:"column:api_key"`
	Status string    `gorm:"column:status"`
}

// APIKeyAuth validates the X-API-Key header against the apps table.
func APIKeyAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			response.Error(c, http.StatusUnauthorized, apperrors.Unauthorized, "X-API-Key header is required")
			c.Abort()
			return
		}

		var app App
		if err := db.Where("api_key = ? AND status = ?", apiKey, "active").First(&app).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.Error(c, http.StatusUnauthorized, apperrors.Unauthorized, "Invalid API key")
			} else {
				response.Error(c, http.StatusInternalServerError, apperrors.InternalError, "Authentication service unavailable")
			}
			c.Abort()
			return
		}

		c.Set(ContextAppID, app.ID)
		c.Next()
	}
}
