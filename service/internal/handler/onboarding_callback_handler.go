package handler

import (
	"io"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/internal/service"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/logger"
	"github.com/hydra/pay-service/pkg/response"
)

type OnboardingCallbackHandler struct {
	db             *gorm.DB
	onboardingRepo *repository.OnboardingRepository
	cfg            *config.Config
}

func NewOnboardingCallbackHandler(db *gorm.DB, cfg *config.Config) *OnboardingCallbackHandler {
	return &OnboardingCallbackHandler{
		db:             db,
		onboardingRepo: repository.NewOnboardingRepository(db),
		cfg:            cfg,
	}
}

func (h *OnboardingCallbackHandler) Callback(c *gin.Context) {
	channelName := c.Param("channel")
	if channelName == "" {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "channel is required")
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "failed to read request body")
		return
	}

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

	adapter, err := service.GetAdapter(channelName, h.cfg)
	if err != nil {
		logger.Warn(c.Request.Context(), "unsupported channel", "channel", channelName)
		response.Error(c, http.StatusBadRequest, errors.ChannelError, "unsupported channel: "+channelName)
		return
	}

	provider, ok := adapter.(channel.OnboardingProvider)
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.ChannelError, "channel does not support onboarding: "+channelName)
		return
	}

	callbackData := &channel.CallbackData{
		RawBody: rawBody,
		Headers: headers,
	}

	result, err := provider.VerifyOnboardingCallback(c.Request.Context(), callbackData)
	if err != nil {
		logger.Error(c.Request.Context(), "callback verification failed", "channel", channelName, "error", err)
		response.Error(c, http.StatusBadRequest, errors.InvalidSignature, err.Error())
		return
	}

	ob, err := h.onboardingRepo.GetByApplymentID(channelName, result.ApplymentID)
	if err != nil {
		logger.Error(c.Request.Context(), "onboarding record not found", "channel", channelName, "applyment_id", result.ApplymentID)
		response.Error(c, http.StatusNotFound, errors.NotFound, "onboarding record not found")
		return
	}

	switch result.Status {
	case model.OnboardingStatusApproved:
		if err := h.onboardingRepo.MarkApproved(ob.ID, result.SubMerchantID); err != nil {
			logger.Error(c.Request.Context(), "failed to mark approved", "error", err)
		}
		h.autoUpdateMerchant(ob.MerchantID, channelName, result.SubMerchantID)
	case model.OnboardingStatusRejected:
		if err := h.onboardingRepo.MarkRejected(ob.ID, result.RejectReason); err != nil {
			logger.Error(c.Request.Context(), "failed to mark rejected", "error", err)
		}
	default:
		if err := h.onboardingRepo.UpdateStatus(ob.ID, result.Status); err != nil {
			logger.Error(c.Request.Context(), "failed to update status", "error", err)
		}
	}

	// Return channel-specific response
	if channelName == model.ChannelAlipay {
		c.String(http.StatusOK, "success")
	} else {
		response.Success(c, gin.H{"code": "SUCCESS", "message": "ok"})
	}
}

func (h *OnboardingCallbackHandler) autoUpdateMerchant(merchantID uuid.UUID, channel, subMerchantID string) {
	if subMerchantID == "" {
		return
	}

	var field string
	switch channel {
	case model.ChannelAlipay:
		field = "alipay_pid"
	case model.ChannelWechat:
		field = "wechat_sub_mchid"
	default:
		return
	}

	if err := h.db.Model(&model.Merchant{}).Where("id = ?", merchantID).Update(field, subMerchantID).Error; err != nil {
		logger.Error(context.Background(), "failed to auto-update merchant", "merchant_id", merchantID, "field", field, "error", err)
	} else {
		logger.Info(context.Background(), "auto-updated merchant", "merchant_id", merchantID, "field", field, "value", subMerchantID)
	}
}
