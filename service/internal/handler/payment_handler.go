package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/middleware"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/internal/service"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/response"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
}

func NewPaymentHandler(db *gorm.DB, cfg *config.Config) *PaymentHandler {
	repo := repository.NewPaymentRepository(db)
	return &PaymentHandler{
		paymentService: service.NewPaymentService(repo, cfg.Wall.WebhookURL),
	}
}

// CreatePayment handles POST /v1/payments/create
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	var req struct {
		UserID      string                 `json:"user_id"`
		PlanID      string                 `json:"plan_id"`
		Amount      int64                  `json:"amount"`
		Currency    string                 `json:"currency"`
		Channel     string                 `json:"channel"`
		Description string                 `json:"description"`
		SuccessURL  string                 `json:"success_url"`
		CancelURL   string                 `json:"cancel_url"`
		Metadata    map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "Invalid request body: "+err.Error())
		return
	}

	if req.UserID == "" {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "user_id is required")
		return
	}
	if req.Amount <= 0 {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "amount must be positive")
		return
	}
	if req.Channel == "" {
		req.Channel = "alipay"
	}

	result, err := h.paymentService.CreatePayment(c.Request.Context(), &service.CreatePaymentInput{
		AppID:       appID.(uuid.UUID),
		UserID:      req.UserID,
		PlanID:      req.PlanID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		ChannelName: req.Channel,
		Description: req.Description,
		SuccessURL:  req.SuccessURL,
		CancelURL:   req.CancelURL,
		Metadata:    req.Metadata,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, gin.H{
		"payment_id":  result.Payment.ID.String(),
		"channel":     result.Payment.Channel,
		"amount":      result.Payment.Amount,
		"currency":    result.Payment.Currency,
		"status":      result.Payment.Status,
		"payment_url": result.PaymentURL,
		"qr_code_url": result.QRCodeURL,
	})
}

// GetPayment handles GET /v1/payments/:id
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "Invalid payment ID")
		return
	}

	payment, err := h.paymentService.GetPayment(appID.(uuid.UUID), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, gin.H{
		"id":          payment.ID.String(),
		"app_id":      payment.AppID.String(),
		"user_id":     payment.UserID,
		"plan_id":     payment.PlanID,
		"amount":      payment.Amount,
		"currency":    payment.Currency,
		"channel":     payment.Channel,
		"status":      payment.Status,
		"external_id": payment.ExternalID,
		"description": payment.Description,
		"created_at":  payment.CreatedAt,
		"paid_at":     payment.PaidAt,
	})
}

// Callback handles POST /v1/payments/callback — unified channel callback endpoint.
func (h *PaymentHandler) Callback(c *gin.Context) {
	channelName := c.Param("channel")
	if channelName == "" {
		channelName = "alipay"
	}

	var req struct {
		PaymentID   string `json:"payment_id"`
		ChannelTxID string `json:"channel_tx_id"`
		Status      string `json:"status"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "Invalid request body: "+err.Error())
		return
	}

	body, _ := json.Marshal(req)

	err := h.paymentService.HandleCallback(c.Request.Context(), channelName, &channel.CallbackData{
		PaymentID:   req.PaymentID,
		ChannelTxID: req.ChannelTxID,
		Status:      req.Status,
		Amount:      req.Amount,
		Currency:    req.Currency,
		RawBody:     body,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "callback processed"})
}

// handleServiceError converts service errors to HTTP responses.
func handleServiceError(c *gin.Context, err error) {
	appErr, ok := err.(*errors.AppError)
	if !ok {
		response.Error(c, http.StatusInternalServerError, errors.InternalError, "Internal server error")
		return
	}
	switch appErr.Code {
	case errors.ValidationError:
		response.Error(c, http.StatusBadRequest, appErr.Code, appErr.Message)
	case errors.NotFound:
		response.Error(c, http.StatusNotFound, appErr.Code, appErr.Message)
	case errors.Unauthorized:
		response.Error(c, http.StatusUnauthorized, appErr.Code, appErr.Message)
	case errors.PaymentFailed, errors.ChannelError:
		response.Error(c, http.StatusBadGateway, appErr.Code, appErr.Message)
	case errors.InvalidSignature:
		response.Error(c, http.StatusBadRequest, appErr.Code, appErr.Message)
	default:
		response.Error(c, http.StatusInternalServerError, errors.InternalError, "Internal server error")
	}
}
