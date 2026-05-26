package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/middleware"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/internal/service"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/metrics"
	"github.com/hydra/pay-service/pkg/response"
)

type RefundHandler struct {
	payService *service.PaymentService
	payRepo    *repository.PaymentRepository
	db         *gorm.DB
}

func NewRefundHandler(db *gorm.DB, cfg *config.Config) *RefundHandler {
	payRepo := repository.NewPaymentRepository(db)
	return &RefundHandler{
		payService: service.NewPaymentService(payRepo, cfg, db),
		payRepo:    payRepo,
		db:         db,
	}
}

// CreateRefund handles POST /v1/refunds
func (h *RefundHandler) CreateRefund(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	var req struct {
		TradeNo      string `json:"trade_no"`
		RefundAmount int64  `json:"refund_amount"` // in cents
		RefundReason string `json:"refund_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid request body: "+err.Error())
		return
	}
	if req.TradeNo == "" {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "trade_no is required")
		return
	}
	if req.RefundAmount <= 0 {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "refund_amount must be positive (in cents)")
		return
	}

	// Verify the payment belongs to this app
	payment, err := h.payRepo.GetByTradeNo(req.TradeNo)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "payment not found")
		return
	}
	if payment.AppID != appID.(uuid.UUID) {
		response.Error(c, http.StatusNotFound, errors.NotFound, "payment not found")
		return
	}

	result, err := h.payService.Refund(c.Request.Context(), &service.RefundInput{
		TradeNo:      req.TradeNo,
		RefundAmount: req.RefundAmount,
		RefundReason: req.RefundReason,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	metrics.RefundsCreatedTotal.WithLabelValues(result.Refund.Channel, "success").Inc()

	response.Success(c, gin.H{
		"refund_id":         result.Refund.ID.String(),
		"status":            result.Refund.Status,
		"refund_amount":     result.Refund.RefundAmount,
		"channel_refund_id": result.Refund.ChannelRefundID,
	})
}

// GetRefund handles GET /v1/refunds/:id
func (h *RefundHandler) GetRefund(c *gin.Context) {
	_, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid refund ID")
		return
	}

	refundRepo := repository.NewRefundRepository(h.db)
	refund, err := refundRepo.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "refund not found")
		return
	}

	response.Success(c, gin.H{
		"refund_id":         refund.ID.String(),
		"payment_id":        refund.PaymentID.String(),
		"trade_no":          refund.TradeNo,
		"channel":           refund.Channel,
		"refund_amount":     refund.RefundAmount,
		"refund_reason":     refund.RefundReason,
		"out_request_no":    refund.OutRequestNo,
		"status":            refund.Status,
		"channel_refund_id": refund.ChannelRefundID,
		"refund_fee":        refund.RefundFee,
		"error_msg":         refund.ErrorMsg,
		"created_at":        refund.CreatedAt,
	})
}

// ListPaymentRefunds handles GET /v1/payments/:id/refunds
func (h *RefundHandler) ListPaymentRefunds(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid payment ID")
		return
	}

	payment, err := h.payRepo.GetByID(paymentID)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "payment not found")
		return
	}
	if payment.AppID != appID.(uuid.UUID) {
		response.Error(c, http.StatusNotFound, errors.NotFound, "payment not found")
		return
	}

	refundRepo := repository.NewRefundRepository(h.db)
	refunds, err := refundRepo.ListByPayment(paymentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errors.InternalError, "failed to list refunds")
		return
	}

	type refundItem struct {
		ID              string `json:"refund_id"`
		TradeNo         string `json:"trade_no"`
		Channel         string `json:"channel"`
		RefundAmount    int64  `json:"refund_amount"`
		RefundReason    string `json:"refund_reason"`
		OutRequestNo    string `json:"out_request_no"`
		Status          string `json:"status"`
		ChannelRefundID string `json:"channel_refund_id"`
		RefundFee       int64  `json:"refund_fee"`
		ErrorMsg        string `json:"error_msg"`
		CreatedAt       string `json:"created_at"`
	}

	items := make([]refundItem, 0, len(refunds))
	for _, r := range refunds {
		items = append(items, refundItem{
			ID:              r.ID.String(),
			TradeNo:         r.TradeNo,
			Channel:         r.Channel,
			RefundAmount:    r.RefundAmount,
			RefundReason:    r.RefundReason,
			OutRequestNo:    r.OutRequestNo,
			Status:          r.Status,
			ChannelRefundID: r.ChannelRefundID,
			RefundFee:       r.RefundFee,
			ErrorMsg:        r.ErrorMsg,
			CreatedAt:       r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	response.Success(c, gin.H{
		"payment_id": payment.ID.String(),
		"refunds":    items,
	})
}
