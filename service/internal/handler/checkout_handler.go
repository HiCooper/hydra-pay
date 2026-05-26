package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/internal/service"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/response"
)

type CheckoutHandler struct {
	sessionRepo *repository.CheckoutSessionRepository
	paymentRepo *repository.PaymentRepository
	payService  *service.PaymentService
	db          *gorm.DB
}

func NewCheckoutHandler(db *gorm.DB, cfg *config.Config) *CheckoutHandler {
	return &CheckoutHandler{
		sessionRepo: repository.NewCheckoutSessionRepository(db),
		paymentRepo: repository.NewPaymentRepository(db),
		payService:  service.NewPaymentService(repository.NewPaymentRepository(db), cfg, db),
		db:          db,
	}
}

// GetCheckout handles GET /api/checkout/:session_id — public.
func (h *CheckoutHandler) GetCheckout(c *gin.Context) {
	id, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid session ID")
		return
	}

	session, err := h.sessionRepo.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "checkout session not found")
		return
	}

	// Look up merchant name from the app
	var app model.App
	merchantName := ""
	if err := h.db.Where("id = ?", session.AppID).First(&app).Error; err == nil {
		merchantName = app.Name
	}

	// For expired or completed sessions, return a friendly status page payload
	// instead of an error, matching Stripe's UX.
	if session.Status != model.CheckoutSessionOpen {
		response.Success(c, gin.H{
			"status":        session.Status,
			"merchant_name": merchantName,
			"cancel_url":    session.CancelURL,
		})
		return
	}

	response.Success(c, gin.H{
		"session_id":    session.ID.String(),
		"amount":        session.Amount,
		"currency":      session.Currency,
		"description":   session.Description,
		"status":        session.Status,
		"success_url":   session.SuccessURL,
		"cancel_url":    session.CancelURL,
		"expires_at":    session.ExpiresAt,
		"merchant_name": merchantName,
	})
}

// ActivatePayment handles POST /api/checkout/:session_id/activate — public.
// Creates a payment and activates the chosen channel.
func (h *CheckoutHandler) ActivatePayment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid session ID")
		return
	}

	var req struct {
		Channel string `json:"channel"`
		UserID  string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid request body")
		return
	}
	if req.Channel != model.ChannelAlipay && req.Channel != model.ChannelWechat {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "channel must be alipay or wechat")
		return
	}

	session, err := h.sessionRepo.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "checkout session not found")
		return
	}

	if session.Status != model.CheckoutSessionOpen {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "session already used or expired")
		return
	}

	if session.ExpiresAt.Before(time.Now()) {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "checkout session has expired")
		return
	}

	// Create payment and activate channel in one step
	result, err := h.payService.CreatePayment(c.Request.Context(), &service.CreatePaymentInput{
		AppID:       session.AppID,
		UserID:      req.UserID,
		Amount:      session.Amount,
		Currency:    session.Currency,
		ChannelName: req.Channel,
		Description: session.Description,
		SuccessURL:  session.SuccessURL,
		CancelURL:   session.CancelURL,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Link session to payment
	h.sessionRepo.MarkCompleted(session.ID, result.Payment.ID)

	response.Success(c, gin.H{
		"payment_id":  result.Payment.ID.String(),
		"channel":     req.Channel,
		"payment_url": result.PaymentURL,
		"qr_code_url": result.QRCodeURL,
	})
}

// GetPaymentStatus handles GET /api/checkout/:session_id/payment-status — public polling.
func (h *CheckoutHandler) GetPaymentStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid session ID")
		return
	}

	session, err := h.sessionRepo.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "checkout session not found")
		return
	}

	if session.PaymentID == nil {
		response.Success(c, gin.H{"status": "pending"})
		return
	}

	payment, err := h.paymentRepo.GetByID(*session.PaymentID)
	if err != nil {
		response.Success(c, gin.H{"status": "pending"})
		return
	}

	response.Success(c, gin.H{
		"status":  payment.Status,
		"paid_at": payment.PaidAt,
	})
}
