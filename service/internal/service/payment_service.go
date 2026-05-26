package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/channel/alipay"
	"github.com/hydra/pay-service/internal/channel/wechat"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/logger"
	"github.com/hydra/pay-service/pkg/tradeno"
	"github.com/hydra/pay-service/pkg/webhook"
)

var webhookClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: false,
	},
}

type PaymentService struct {
	repo              *repository.PaymentRepository
	taskRepo          *repository.ScheduledTaskRepository
	refundRepo        *repository.RefundRepository
	wallWebhookURL    string
	wallWebhookSecret string
	cfg               *config.Config
	db                *gorm.DB
	webhookSem        chan struct{}
}

func (s *PaymentService) GetConfig() *config.Config { return s.cfg }

func NewPaymentService(repo *repository.PaymentRepository, cfg *config.Config, db *gorm.DB) *PaymentService {
	poolSize := cfg.Server.WebhookPoolSize
	if poolSize <= 0 {
		poolSize = 10
	}
	return &PaymentService{
		repo:              repo,
		taskRepo:          repository.NewScheduledTaskRepository(db),
		refundRepo:        repository.NewRefundRepository(db),
		wallWebhookURL:    cfg.Wall.WebhookURL,
		wallWebhookSecret: cfg.Wall.WebhookSecret,
		cfg:               cfg,
		db:                db,
		webhookSem:        make(chan struct{}, poolSize),
	}
}

type CreatePaymentInput struct {
	AppID        uuid.UUID
	UserID       string
	PlanID       string
	Amount       int64
	Currency     string
	ChannelName  string
	TradeType    string
	Description  string
	SuccessURL   string
	CancelURL    string
	OpenID         string
	ChannelAppID   string
	SubMerchantID   string
	SubChannelAppID string
	ClientIP        string
	NotifyURL       string
	Metadata       map[string]interface{}
}

type CreatePaymentResult struct {
	Payment    *model.Payment
	PaymentURL string
	QRCodeURL  string
}

type RefundInput struct {
	TradeNo      string
	RefundAmount int64  // in cents
	RefundReason string
}

type RefundResult struct {
	Refund  *model.Refund
	Payment *model.Payment
}

// Refund creates a refund for a paid payment.
func (s *PaymentService) Refund(ctx context.Context, input *RefundInput) (*RefundResult, error) {
	if input == nil || input.TradeNo == "" {
		return nil, errors.New(errors.ValidationError, "trade_no is required")
	}
	if input.RefundAmount <= 0 {
		return nil, errors.New(errors.ValidationError, "refund_amount must be positive")
	}

	payment, err := s.repo.GetByTradeNo(input.TradeNo)
	if err != nil {
		return nil, errors.New(errors.NotFound, "payment not found")
	}
	if payment.Status != model.PaymentStatusPaid {
		return nil, errors.New(errors.ValidationError, "only paid orders can be refunded")
	}

	if input.RefundAmount > payment.Amount {
		return nil, errors.New(errors.ValidationError, "refund amount exceeds payment amount")
	}

	outReqNo := "RF" + payment.TradeNo

	// Idempotency: check if refund already exists
	if existing, err := s.refundRepo.GetByOutRequestNo(outReqNo); err == nil {
		logger.Info(ctx, "refund already exists", "out_request_no", outReqNo, "status", existing.Status)
		return &RefundResult{Refund: existing, Payment: payment}, nil
	}

	refund := &model.Refund{
		PaymentID:    payment.ID,
		AppID:        payment.AppID,
		TradeNo:      payment.TradeNo,
		Channel:      payment.Channel,
		RefundAmount: input.RefundAmount,
		RefundReason: input.RefundReason,
		OutRequestNo: outReqNo,
		Status:       model.RefundStatusProcessing,
	}
	if err := s.refundRepo.Create(refund); err != nil {
		return nil, errors.Wrap(errors.InternalError, "failed to create refund record", err)
	}

	adapter, err := GetAdapter(payment.Channel, s.cfg)
	if err != nil {
			s.refundRepo.UpdateStatus(refund.ID, model.RefundStatusFailed, "", 0, err.Error())
		repository.RecordEvent(s.db, model.EventRefund, payment.Channel,
			payment.ID, "", nil, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "failed to init channel adapter", err)
	}

	chResp, err := adapter.Refund(ctx, &channel.RefundRequest{
		TradeNo:      payment.TradeNo,
		ChannelTxID:  payment.ExternalID,
		RefundAmount: input.RefundAmount,
		TotalAmount:  payment.Amount,
		RefundReason: input.RefundReason,
		OutRequestNo: outReqNo,
	})
	if err != nil {
			s.refundRepo.UpdateStatus(refund.ID, model.RefundStatusFailed, "", 0, err.Error())
		repository.RecordEvent(s.db, model.EventRefund, payment.Channel,
			payment.ID, "", nil, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "refund failed", err)
	}

	s.refundRepo.UpdateStatus(refund.ID, model.RefundStatusSuccess, chResp.ChannelRefundID, chResp.RefundFee, "")

	respJSON, _ := json.Marshal(chResp.RawResponse)
	refund.Status = model.RefundStatusSuccess
	refund.ChannelRefundID = chResp.ChannelRefundID
	refund.RefundFee = chResp.RefundFee
	refund.ResponseData = respJSON

	if err := s.repo.UpdateStatus(payment.ID, model.PaymentStatusRefunded, payment.ExternalID); err != nil {
		logger.Error(ctx, "failed to update payment status to refunded", "error", err)
	}
	payment.Status = model.PaymentStatusRefunded

	repository.RecordEvent(s.db, model.EventRefund, payment.Channel,
		payment.ID, string(respJSON),
		map[string]interface{}{
			"refund_fee":  chResp.RefundFee,
			"refund_id":   chResp.ChannelRefundID,
		}, "")

	s.safeNotifyWallRefund(payment, refund)

	return &RefundResult{Refund: refund, Payment: payment}, nil
}

func GetAdapter(name string, cfg *config.Config) (channel.Adapter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	switch name {
	case model.ChannelAlipay:
		return alipay.NewAdapter(&cfg.Alipay)
	case model.ChannelWechat:
		return wechat.NewAdapter(&cfg.Wechat)
	default:
		return nil, fmt.Errorf("unsupported channel: %s", name)
	}
}

func (s *PaymentService) CreatePayment(ctx context.Context, input *CreatePaymentInput) (*CreatePaymentResult, error) {
	if input == nil {
		return nil, errors.New(errors.ValidationError, "payment input cannot be nil")
	}

	if input.Currency == "" {
		input.Currency = "CNY"
	}

	var meta datatypes.JSON
	if input.Metadata != nil {
		meta = datatypes.JSON(marshalOrEmpty(input.Metadata))
	}

	tradeNo := tradeno.Generate(input.ChannelName)

	payment := &model.Payment{
		TradeNo:     tradeNo,
		AppID:       input.AppID,
		UserID:      input.UserID,
		PlanID:      input.PlanID,
		Amount:      input.Amount,
		Currency:    input.Currency,
		Channel:     input.ChannelName,
		Status:      model.PaymentStatusPending,
		Description: input.Description,
		SuccessURL:  input.SuccessURL,
		CancelURL:   input.CancelURL,
		Metadata:    meta,
	}

	if err := s.repo.Create(payment); err != nil {
		return nil, errors.Wrap(errors.InternalError, "failed to create payment", err)
	}

	repository.RecordEvent(s.db, model.EventCreated, input.ChannelName,
		payment.ID, "", nil, "")

	// Schedule timeout check
	if err := s.taskRepo.Create(&model.ScheduledTask{
		TaskType:    model.TaskTypeOrderTimeout,
		ReferenceID: payment.ID,
		ExecuteAt:   payment.CreatedAt.Add(15 * time.Minute),
		Status:      model.TaskStatusPending,
	}); err != nil {
		logger.Error(ctx, "failed to schedule timeout task", "trade_no", tradeNo, "error", err)
	}

	// If channel is specified, activate immediately
	if input.ChannelName != "" {
		result, err := s.ActivateChannel(ctx, payment, input)
		if err != nil {
			s.repo.UpdateStatus(payment.ID, model.PaymentStatusFailed, "")
			repository.RecordEvent(s.db, model.EventChannelRequest, input.ChannelName,
				payment.ID, "", nil, err.Error())
			return nil, err
		}
		return result, nil
	}

	return &CreatePaymentResult{Payment: payment}, nil
}

// ActivateChannel activates a pending payment with the given channel.
func (s *PaymentService) ActivateChannel(ctx context.Context, payment *model.Payment, input *CreatePaymentInput) (*CreatePaymentResult, error) {
	adapter, err := GetAdapter(input.ChannelName, s.cfg)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "failed to init channel adapter", err)
	}

	chResp, err := adapter.CreatePayment(ctx, &channel.CreatePaymentRequest{
		PaymentID:      payment.TradeNo,
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		Description:    payment.Description,
		SuccessURL:     input.SuccessURL,
		CancelURL:      input.CancelURL,
		TradeType:      input.TradeType,
		OpenID:         input.OpenID,
		ChannelAppID:   input.ChannelAppID,
		SubMerchantID:  input.SubMerchantID,
		SubChannelAppID: input.SubChannelAppID,
		ClientIP:        input.ClientIP,
		NotifyURL:       input.NotifyURL,
	})
	if err != nil {
		return nil, errors.Wrap(errors.PaymentFailed, "channel payment creation failed", err)
	}

	repository.RecordEvent(s.db, model.EventChannelRequest, input.ChannelName,
		payment.ID, "", chResp.RawResponse, "")

	if updateErr := s.repo.UpdateStatus(payment.ID, model.PaymentStatusProcessing, chResp.ChannelTxID); updateErr != nil {
		logger.Error(ctx, "failed to update payment status to processing", "error", updateErr)
	}
	if updateErr := s.repo.UpdateChannelURLs(payment.ID, chResp.PaymentURL, chResp.QRCodeURL); updateErr != nil {
		logger.Error(ctx, "failed to update payment URLs", "error", updateErr)
	}
	if updateErr := s.repo.UpdateChannel(payment.ID, input.ChannelName); updateErr != nil {
		logger.Error(ctx, "failed to update payment channel", "error", updateErr)
	}

	payment.Status = model.PaymentStatusProcessing
	payment.ExternalID = chResp.ChannelTxID
	payment.Channel = input.ChannelName
	payment.PaymentURL = chResp.PaymentURL
	payment.QRCodeURL = chResp.QRCodeURL

	return &CreatePaymentResult{
		Payment:    payment,
		PaymentURL: chResp.PaymentURL,
		QRCodeURL:  chResp.QRCodeURL,
	}, nil
}

func (s *PaymentService) GetPayment(appID, id uuid.UUID) (*model.Payment, error) {
	payment, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.NotFound, "payment not found")
		}
		return nil, errors.Wrap(errors.InternalError, "failed to get payment", err)
	}
	if payment.AppID != appID {
		return nil, errors.New(errors.NotFound, "payment not found")
	}
	return payment, nil
}

func (s *PaymentService) ListPayments(appID uuid.UUID, page, pageSize int) ([]model.Payment, int64, error) {
	return s.repo.ListByApp(appID, page, pageSize)
}

// HandleCallback verifies the callback with the channel adapter and updates the payment.
func (s *PaymentService) HandleCallback(ctx context.Context, channelName string, data *channel.CallbackData) (*channel.CallbackResult, error) {
	// Record raw callback arrival before any processing
	logger.Info(ctx, "callback received", "channel", channelName, "body_len", len(data.RawBody))

	adapter, err := GetAdapter(channelName, s.cfg)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "failed to init channel adapter", err)
	}

	result, err := adapter.VerifyCallback(ctx, data)
	if err != nil {
		logger.Error(ctx, "callback verification failed", "channel", channelName, "error", err)
		return nil, errors.Wrap(errors.InvalidSignature, "callback verification failed", err)
	}

	// Deduplicate by channel notification ID
	if isDuplicate := checkDedup(s.db, result); isDuplicate {
		logger.Info(ctx, "duplicate callback ignored", "channel", channelName, "payment_id", result.PaymentID)
		return result, nil
	}

	tradeNo := result.PaymentID

	payment, err := s.repo.GetByTradeNo(tradeNo)
	if err != nil {
		return nil, errors.New(errors.NotFound, "payment not found for trade_no: "+tradeNo)
	}

	// Persist channel-specific callback record
	saveCallback(s.db, payment.ID, result)

	// Cancel scheduled timeout task
	s.taskRepo.CancelByReference(payment.ID)

	if payment.Status == model.PaymentStatusPaid || payment.Status == model.PaymentStatusRefunded {
		logger.Info(ctx, "callback ignored, already in terminal state", "trade_no", tradeNo, "status", payment.Status)
		return result, nil
	}

	fromStatus := payment.Status

	switch result.Status {
	case model.PaymentStatusPaid:
		applied, err := s.repo.MarkPaidIfPending(payment.ID, result.ChannelTxID)
		if err != nil {
			return nil, err
		}
		if !applied {
			logger.Warn(ctx, "callback race, already updated by concurrent callback", "trade_no", tradeNo)
			return result, nil
		}
		payment.Status = model.PaymentStatusPaid
	case model.PaymentStatusFailed:
		if err := s.repo.UpdateStatus(payment.ID, model.PaymentStatusFailed, result.ChannelTxID); err != nil {
			return nil, err
		}
		payment.Status = model.PaymentStatusFailed
	default:
		logger.Info(ctx, "callback received non-terminal status", "status", result.Status, "trade_no", tradeNo)
		return result, nil
	}

	// Record status transition
	repository.RecordEvent(s.db, model.EventStatusChanged, channelName,
		payment.ID, "",
		map[string]interface{}{
			"from": fromStatus,
			"to":   payment.Status,
		}, "")

	s.safeNotifyWall(payment, result)

	return result, nil
}

func (s *PaymentService) safeNotifyWall(payment *model.Payment, result *channel.CallbackResult) {
	select {
	case s.webhookSem <- struct{}{}:
		go func() {
			defer func() { <-s.webhookSem }()
			s.doSafeNotifyWall(payment, result)
		}()
	default:
		logger.Error(context.Background(), "webhook pool exhausted, dropping notification",
			"payment_id", payment.ID, "status", payment.Status)
		repository.RecordEvent(s.db, model.EventWebhookSent, payment.Channel,
			payment.ID, "", nil, "dropped: webhook pool exhausted")
	}
}

func (s *PaymentService) doSafeNotifyWall(payment *model.Payment, result *channel.CallbackResult) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(context.Background(), "panic in webhook notification", "error", r)
		}
	}()
	s.notifyWall(payment, result)
}

func (s *PaymentService) notifyWall(payment *model.Payment, result *channel.CallbackResult) {
	// Look up the app's webhook URL and secret; fall back to global config
	webhookURL := s.wallWebhookURL
	webhookSecret := s.wallWebhookSecret
	var app model.App
	if err := s.db.First(&app, "id = ?", payment.AppID).Error; err == nil {
		if app.WebhookURL != "" {
			webhookURL = app.WebhookURL
		}
		if app.WebhookSecret != "" {
			webhookSecret = app.WebhookSecret
		}
	}
	if webhookURL == "" {
		return
	}

	event := "payment.success"
	if payment.Status == model.PaymentStatusFailed {
		event = "payment.failed"
	}

	payload := map[string]interface{}{
		"event":      event,
		"payment_id": payment.ID.String(),
		"user_id":    payment.UserID,
		"plan_id":    payment.PlanID,
		"amount":     payment.Amount,
		"currency":   payment.Currency,
		"status":     payment.Status,
		"channel":    payment.Channel,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error(context.Background(), "failed to marshal webhook payload", "error", err)
		return
	}

	retries := []time.Duration{1 * time.Second, 5 * time.Second, 15 * time.Second}
	var lastErr error
	for i, delay := range retries {
		if i > 0 {
			time.Sleep(delay)
		}
		if err := s.sendWebhook(webhookURL, body, webhookSecret); err != nil {
			lastErr = err
			logger.Warn(context.Background(), "webhook attempt failed", "attempt", i+1, "url", webhookURL, "error", err)
			continue
		}
		repository.RecordEvent(s.db, model.EventWebhookSent, payment.Channel,
			payment.ID, string(body),
			map[string]interface{}{"attempt": i + 1, "url": webhookURL}, "")
		return
	}
	repository.RecordEvent(s.db, model.EventWebhookSent, payment.Channel,
		payment.ID, string(body), nil,
		fmt.Sprintf("failed after 3 attempts: %v", lastErr))
	logger.Error(context.Background(), "webhook permanently failed after 3 attempts", "error", lastErr)
}

func (s *PaymentService) safeNotifyWallRefund(payment *model.Payment, refund *model.Refund) {
	select {
	case s.webhookSem <- struct{}{}:
		go func() {
			defer func() { <-s.webhookSem }()
			s.doSafeNotifyWallRefund(payment, refund)
		}()
	default:
		logger.Error(context.Background(), "webhook pool exhausted, dropping refund notification",
			"payment_id", payment.ID, "refund_id", refund.ID)
		repository.RecordEvent(s.db, model.EventWebhookSent, payment.Channel,
			payment.ID, "", nil, "dropped: webhook pool exhausted")
	}
}

func (s *PaymentService) doSafeNotifyWallRefund(payment *model.Payment, refund *model.Refund) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(context.Background(), "panic in refund webhook notification", "error", r)
		}
	}()
	s.notifyWallRefund(payment, refund)
}

func (s *PaymentService) notifyWallRefund(payment *model.Payment, refund *model.Refund) {
	webhookURL := s.wallWebhookURL
	webhookSecret := s.wallWebhookSecret
	var app model.App
	if err := s.db.First(&app, "id = ?", payment.AppID).Error; err == nil {
		if app.WebhookURL != "" {
			webhookURL = app.WebhookURL
		}
		if app.WebhookSecret != "" {
			webhookSecret = app.WebhookSecret
		}
	}
	if webhookURL == "" {
		return
	}

	payload := map[string]interface{}{
		"event":         "payment.refunded",
		"payment_id":    payment.ID.String(),
		"user_id":       payment.UserID,
		"plan_id":       payment.PlanID,
		"amount":        payment.Amount,
		"currency":      payment.Currency,
		"status":        payment.Status,
		"channel":       payment.Channel,
		"refund_amount": refund.RefundAmount,
		"refund_reason": refund.RefundReason,
		"refund_id":     refund.ID.String(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error(context.Background(), "failed to marshal refund webhook payload", "error", err)
		return
	}

	retries := []time.Duration{1 * time.Second, 5 * time.Second, 15 * time.Second}
	var lastErr error
	for i, delay := range retries {
		if i > 0 {
			time.Sleep(delay)
		}
		if err := s.sendWebhook(webhookURL, body, webhookSecret); err != nil {
			lastErr = err
			logger.Warn(context.Background(), "refund webhook attempt failed", "attempt", i+1, "url", webhookURL, "error", err)
			continue
		}
		repository.RecordEvent(s.db, model.EventWebhookSent, payment.Channel,
			payment.ID, string(body),
			map[string]interface{}{"attempt": i + 1, "url": webhookURL}, "")
		return
	}
	repository.RecordEvent(s.db, model.EventWebhookSent, payment.Channel,
		payment.ID, string(body), nil,
		fmt.Sprintf("failed after 3 attempts: %v", lastErr))
	logger.Error(context.Background(), "refund webhook permanently failed after 3 attempts", "error", lastErr)
}

func (s *PaymentService) sendWebhook(url string, body []byte, secret string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if secret != "" {
		sig := webhook.Sign(secret, body, time.Now().Unix())
		req.Header.Set("X-HydraPay-Signature", sig)
	}

	resp, err := webhookClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// checkDedup returns true if this callback notification has already been processed.
func checkDedup(db *gorm.DB, result *channel.CallbackResult) bool {
	if result.AlipayCallback != nil && result.AlipayCallback.NotifyID != "" {
		var count int64
		db.Model(&model.AlipayCallback{}).Where("notify_id = ?", result.AlipayCallback.NotifyID).Count(&count)
		return count > 0
	}
	if result.WechatPayCallback != nil && result.WechatPayCallback.NotificationID != "" {
		var count int64
		db.Model(&model.WechatPayCallback{}).Where("notification_id = ?", result.WechatPayCallback.NotificationID).Count(&count)
		return count > 0
	}
	return false
}

// saveCallback persists the channel-specific callback record.
func saveCallback(db *gorm.DB, paymentID uuid.UUID, result *channel.CallbackResult) {
	if result.AlipayCallback != nil {
		result.AlipayCallback.PaymentID = paymentID
		if err := db.Create(result.AlipayCallback).Error; err != nil {
			logger.Error(context.Background(), "failed to save alipay callback", "error", err)
		}
	}
	if result.WechatPayCallback != nil {
		result.WechatPayCallback.PaymentID = paymentID
		if err := db.Create(result.WechatPayCallback).Error; err != nil {
			logger.Error(context.Background(), "failed to save wechat callback", "error", err)
		}
	}
}

func marshalOrEmpty(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}