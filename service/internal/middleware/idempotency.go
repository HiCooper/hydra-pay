package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/pkg/logger"
)

// responseCapture wraps gin.ResponseWriter to capture the response body and status.
type responseCapture struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func newResponseCapture(w gin.ResponseWriter) *responseCapture {
	return &responseCapture{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		status:         http.StatusOK,
	}
}

func (r *responseCapture) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseCapture) Write(data []byte) (int, error) {
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}

// Idempotency provides idempotent request replay for POST/PUT/PATCH endpoints.
// It reads the Idempotency-Key header, checks for a cached response, and either
// replays the cached response or captures the current one for future replay.
// Must be placed AFTER APIKeyAuth middleware (needs app_id from context).
func Idempotency(db *gorm.DB) gin.HandlerFunc {
	repo := repository.NewIdempotencyRepository(db)

	return func(c *gin.Context) {
		// Only apply to mutating methods
		method := c.Request.Method
		if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
			c.Next()
			return
		}

		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		appIDVal, exists := c.Get(ContextAppID)
		if !exists {
			c.Next()
			return
		}
		appID := appIDVal.(uuid.UUID)

		// Check for existing cached response
		if existing, err := repo.FindByKey(appID, key); err == nil {
			logger.Info(c.Request.Context(), "replaying cached response", "key", key, "status", existing.ResponseStatus)
			c.Data(existing.ResponseStatus, "application/json; charset=utf-8", []byte(existing.ResponseBody))
			c.Abort()
			return
		}

		// Wrap writer to capture response
		capture := newResponseCapture(c.Writer)
		c.Writer = capture

		c.Next()

		// Only cache successful responses (2xx)
		if capture.status >= 200 && capture.status < 300 {
			record := &model.IdempotencyRecord{
				IdempotencyKey: key,
				AppID:          appID,
				ResponseStatus: capture.status,
				ResponseBody:   capture.body.String(),
				ExpiresAt:      time.Now().Add(24 * time.Hour),
			}
			if err := repo.Create(record); err != nil {
				logger.Error(c.Request.Context(), "failed to store idempotency record", "error", err)
			}
		}
	}
}
