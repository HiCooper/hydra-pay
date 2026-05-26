package handler

import (
	"io"
	"log"
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
		log.Printf("[onboarding] unsupported channel: %s", channelName)
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
		log.Printf("[onboarding] callback verification failed: channel=%s, err=%v", channelName, err)
		response.Error(c, http.StatusBadRequest, errors.InvalidSignature, err.Error())
		return
	}

	ob, err := h.onboardingRepo.GetByApplymentID(channelName, result.ApplymentID)
	if err != nil {
		log.Printf("[onboarding] onboarding record not found: channel=%s, applyment_id=%s", channelName, result.ApplymentID)
		response.Error(c, http.StatusNotFound, errors.NotFound, "onboarding record not found")
		return
	}

	switch result.Status {
	case model.OnboardingStatusApproved:
		if err := h.onboardingRepo.MarkApproved(ob.ID, result.SubMerchantID); err != nil {
			log.Printf("[onboarding] failed to mark approved: %v", err)
		}
		// Auto-update the App with the sub_merchant_id
		h.autoUpdateApp(ob.AppID, channelName, result.SubMerchantID)
	case model.OnboardingStatusRejected:
		if err := h.onboardingRepo.MarkRejected(ob.ID, result.RejectReason); err != nil {
			log.Printf("[onboarding] failed to mark rejected: %v", err)
		}
	default:
		if err := h.onboardingRepo.UpdateStatus(ob.ID, result.Status); err != nil {
			log.Printf("[onboarding] failed to update status: %v", err)
		}
	}

	// Return channel-specific response
	if channelName == model.ChannelAlipay {
		c.String(http.StatusOK, "success")
	} else {
		response.Success(c, gin.H{"code": "SUCCESS", "message": "ok"})
	}
}

func (h *OnboardingCallbackHandler) autoUpdateApp(appID uuid.UUID, channel, subMerchantID string) {
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

	if err := h.db.Model(&model.App{}).Where("id = ?", appID).Update(field, subMerchantID).Error; err != nil {
		log.Printf("[onboarding] failed to auto-update app %s field %s: %v", appID, field, err)
	} else {
		log.Printf("[onboarding] auto-updated app %s %s = %s", appID, field, subMerchantID)
	}
}
