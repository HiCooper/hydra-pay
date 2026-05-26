package portal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/response"
)

const merchantContextKey = "merchant_id"

var signSecret = func() string {
	s := os.Getenv("PORTAL_SIGN_SECRET")
	if s == "" {
		s = "hydra-pay-portal-secret-change-me"
	}
	return s
}()

func signToken(merchantID string) string {
	mac := hmac.New(sha256.New, []byte(signSecret))
	mac.Write([]byte(merchantID))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(merchantID)) + "." + sig
}

func verifyToken(token string) (string, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	idBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(signSecret))
	mac.Write(idBytes)
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return string(idBytes), hmac.Equal([]byte(expected), []byte(parts[1]))
}

// Login handles POST /portal/api/login
func Login(c *gin.Context, db *gorm.DB) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.Email == "" || req.Password == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "email and password are required")
		return
	}

	var m model.Merchant
	if err := db.Where("email = ? AND status = ?", req.Email, "active").First(&m).Error; err != nil {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
		return
	}

	if !m.CheckPassword(req.Password) {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
		return
	}

	token := signToken(m.ID.String())
	response.Success(c, gin.H{
		"token":        token,
		"merchant_id":  m.ID.String(),
		"merchant_name": m.Name,
	})
}

// MerchantAuth is a Gin middleware that validates a Bearer token and sets merchant_id in context.
func MerchantAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "authorization header required")
			c.Abort()
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		idStr, ok := verifyToken(token)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
			c.Abort()
			return
		}

		var m model.Merchant
		if err := db.Where("id = ? AND status = ?", idStr, "active").First(&m).Error; err != nil {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
			c.Abort()
			return
		}

		c.Set(merchantContextKey, m.ID)
		c.Next()
	}
}
