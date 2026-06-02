package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/middleware"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/internal/service"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/metrics"
	"github.com/hydra/pay-service/pkg/response"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
	db             *gorm.DB
}

func NewPaymentHandler(db *gorm.DB, cfg *config.Config) *PaymentHandler {
	repo := repository.NewPaymentRepository(db)
	return &PaymentHandler{
		paymentService: service.NewPaymentService(repo, cfg, db),
		db:             db,
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
		UserID       string                 `json:"user_id"`
		PlanID       string                 `json:"plan_id"`
		Amount       int64                  `json:"amount"`
		Currency     string                 `json:"currency"`
		Channel      string                 `json:"channel"`
		TradeType    string                 `json:"trade_type"`
		SuccessURL   string                 `json:"success_url"`
		CancelURL    string                 `json:"cancel_url"`
		Description  string                 `json:"description"`
		OpenID         string                 `json:"open_id"`
		ChannelAppID   string                 `json:"channel_app_id"`
		SubMerchantID   string                 `json:"sub_merchant_id"`
		SubChannelAppID string                 `json:"sub_channel_app_id"`
		ClientIP        string                 `json:"client_ip"`
		NotifyURL       string                 `json:"notify_url"`
		Metadata     map[string]interface{} `json:"metadata"`
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
	}

	// Auto-resolve SubMerchantID from App → Merchant if not provided
	if req.SubMerchantID == "" && req.Channel != "" {
		var app model.App
		if err := h.db.First(&app, "id = ?", appID).Error; err == nil {
			var merchant model.Merchant
			if err := h.db.First(&merchant, "id = ?", app.MerchantID).Error; err == nil {
				switch req.Channel {
				case model.ChannelAlipay:
					req.SubMerchantID = merchant.AlipayPID
				case model.ChannelWechat:
					req.SubMerchantID = merchant.WechatSubMchid
				case model.ChannelUnionpay:
					req.SubMerchantID = merchant.UnionpaySubMerID
				case model.ChannelEcny:
					req.SubMerchantID = merchant.EcnySubMerID
				}
				if req.SubMerchantID != "" && req.SubChannelAppID == "" {
					req.SubChannelAppID = merchant.WechatSubAppid
				}
			}
		}
	}

	result, err := h.paymentService.CreatePayment(c.Request.Context(), &service.CreatePaymentInput{
		AppID:        appID.(uuid.UUID),
		UserID:       req.UserID,
		PlanID:       req.PlanID,
		Amount:       req.Amount,
		Currency:     req.Currency,
		ChannelName:  req.Channel,
		TradeType:    req.TradeType,
		Description:  req.Description,
		SuccessURL:   req.SuccessURL,
		CancelURL:    req.CancelURL,
		OpenID:         req.OpenID,
		ChannelAppID:   req.ChannelAppID,
		SubMerchantID:  req.SubMerchantID,
		SubChannelAppID: req.SubChannelAppID,
		ClientIP:        req.ClientIP,
		NotifyURL:       req.NotifyURL,
		Metadata:        req.Metadata,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	metrics.PaymentsCreatedTotal.WithLabelValues(result.Payment.Channel, "success").Inc()

	response.Success(c, gin.H{
		"payment_id":  result.Payment.ID.String(),
		"trade_no":    result.Payment.TradeNo,
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
		"trade_no":    payment.TradeNo,
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

// Callback handles POST /v1/payments/callback/:channel — unified channel callback endpoint.
func (h *PaymentHandler) Callback(c *gin.Context) {
	channelName := c.Param("channel")
	if channelName == "" {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "channel is required")
		return
	}

	// Read raw body bytes — each channel has its own format
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "failed to read request body")
		return
	}

	// Collect relevant HTTP headers (needed for WeChat V3 signature verification)
	headers := make(map[string]string)
	for _, key := range []string{
		"Wechatpay-Timestamp",
		"Wechatpay-Nonce",
		"Wechatpay-Signature",
		"Wechatpay-Serial",
		"Content-Type",
	} {
		if v := c.GetHeader(key); v != "" {
			headers[key] = v
		}
	}

	result, err := h.paymentService.HandleCallback(c.Request.Context(), channelName, &channel.CallbackData{
		RawBody: rawBody,
		Headers: headers,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	// Return channel-specific response
	// Alipay and UnionPay expect "success" plain text; WeChat expects JSON
	if channelName == model.ChannelAlipay || channelName == model.ChannelUnionpay || channelName == model.ChannelEcny {
		c.String(http.StatusOK, "success")
	} else {
		response.Success(c, gin.H{"code": "SUCCESS", "message": "ok"})
	}

	_ = result // result is used for logging within the service
}

// handleServiceError converts service errors to HTTP responses.
	// handleServiceError converts service errors to HTTP responses.
	func handleServiceError(c *gin.Context, err error) {
		appErr, ok := err.(*errors.AppError)
		if !ok {
			response.Error(c, http.StatusInternalServerError, errors.InternalError, "Internal server error")
			return
		}
		msg := appErr.Message
		if appErr.Unwrap() != nil {
			msg = appErr.Message + ": " + appErr.Unwrap().Error()
		}
		switch appErr.Code {
		case errors.ValidationError:
			response.Error(c, http.StatusBadRequest, appErr.Code, msg)
		case errors.NotFound:
			response.Error(c, http.StatusNotFound, appErr.Code, msg)
		case errors.Unauthorized:
			response.Error(c, http.StatusUnauthorized, appErr.Code, msg)
		case errors.PaymentFailed, errors.ChannelError:
			response.Error(c, http.StatusBadGateway, appErr.Code, msg)
		case errors.InvalidSignature:
			response.Error(c, http.StatusBadRequest, appErr.Code, msg)
		default:
			response.Error(c, http.StatusInternalServerError, errors.InternalError, msg)
		}
	}
