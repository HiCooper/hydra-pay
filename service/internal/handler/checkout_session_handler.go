package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/middleware"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/response"
)

type CheckoutSessionHandler struct {
	repo *repository.CheckoutSessionRepository
}

func NewCheckoutSessionHandler(db *gorm.DB) *CheckoutSessionHandler {
	return &CheckoutSessionHandler{
		repo: repository.NewCheckoutSessionRepository(db),
	}
}

// CreateSession handles POST /v1/checkout/sessions — authenticated.
func (h *CheckoutSessionHandler) CreateSession(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	var req struct {
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
		Description string `json:"description"`
		SuccessURL  string `json:"success_url"`
		CancelURL   string `json:"cancel_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid request body")
		return
	}
	if req.Amount <= 0 {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "amount must be positive")
		return
	}
	if req.Currency == "" {
		req.Currency = "CNY"
	}

	session := &model.CheckoutSession{
		AppID:       appID.(uuid.UUID),
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		SuccessURL:  req.SuccessURL,
		CancelURL:   req.CancelURL,
		Status:      model.CheckoutSessionOpen,
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	}

	if err := h.repo.Create(session); err != nil {
		response.Error(c, http.StatusInternalServerError, errors.InternalError, "failed to create checkout session")
		return
	}

	response.Success(c, gin.H{
		"id":           session.ID.String(),
		"checkout_url": "/pay/checkout/" + session.ID.String(),
		"expires_at":   session.ExpiresAt,
	})
}
