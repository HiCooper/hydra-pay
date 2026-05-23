package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/pkg/errors"
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
	repo           *repository.PaymentRepository
	wallWebhookURL string
}

func NewPaymentService(repo *repository.PaymentRepository, wallWebhookURL string) *PaymentService {
	return &PaymentService{repo: repo, wallWebhookURL: wallWebhookURL}
}

type CreatePaymentInput struct {
	AppID       uuid.UUID
	UserID      string
	PlanID      string
	Amount      int64
	Currency    string
	ChannelName string
	Description string
	SuccessURL  string
	CancelURL   string
	Metadata    map[string]interface{}
}

type CreatePaymentResult struct {
	Payment    *model.Payment
	PaymentURL string
	QRCodeURL  string
}

func (s *PaymentService) CreatePayment(ctx context.Context, input *CreatePaymentInput) (*CreatePaymentResult, error) {
	adapter, err := channel.GetAdapter(input.ChannelName)
	if err != nil {
		return nil, errors.New(errors.ValidationError, fmt.Sprintf("unsupported channel: %s", input.ChannelName))
	}

	if input.Currency == "" {
		input.Currency = "CNY"
	}

	var meta datatypes.JSON
	if input.Metadata != nil {
		meta = datatypes.JSON(marshalOrEmpty(input.Metadata))
	}

	payment := &model.Payment{
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

	chResp, err := adapter.CreatePayment(ctx, &channel.CreatePaymentRequest{
		PaymentID:   payment.ID.String(),
		Amount:      input.Amount,
		Currency:    input.Currency,
		Description: input.Description,
		SuccessURL:  input.SuccessURL,
		CancelURL:   input.CancelURL,
	})
	if err != nil {
		if updateErr := s.repo.UpdateStatus(payment.ID, model.PaymentStatusFailed, ""); updateErr != nil {
			log.Printf("[pay] failed to update payment status after channel error: %v", updateErr)
		}
		return nil, errors.Wrap(errors.PaymentFailed, "channel payment creation failed", err)
	}

	if updateErr := s.repo.UpdateStatus(payment.ID, model.PaymentStatusProcessing, chResp.ChannelTxID); updateErr != nil {
		log.Printf("[pay] failed to update payment status to processing: %v", updateErr)
	}
	payment.Status = model.PaymentStatusProcessing
	payment.ExternalID = chResp.ChannelTxID

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

func (s *PaymentService) HandleCallback(ctx context.Context, channelName string, data *channel.CallbackData) error {
	adapter, err := channel.GetAdapter(channelName)
	if err != nil {
		return errors.New(errors.ValidationError, fmt.Sprintf("unsupported channel: %s", channelName))
	}

	if err := adapter.VerifyCallback(ctx, data); err != nil {
		return errors.Wrap(errors.InvalidSignature, "callback verification failed", err)
	}

	paymentID, err := uuid.Parse(data.PaymentID)
	if err != nil {
		return errors.New(errors.ValidationError, "invalid payment ID in callback")
	}

	payment, err := s.repo.GetByID(paymentID)
	if err != nil {
		return errors.New(errors.NotFound, "payment not found for callback")
	}

	if payment.Status == model.PaymentStatusPaid || payment.Status == model.PaymentStatusRefunded {
		return nil
	}

	switch data.Status {
	case "paid", "success", "TRADE_SUCCESS":
		if err := s.repo.MarkPaid(paymentID, data.ChannelTxID); err != nil {
			return err
		}
		payment.Status = model.PaymentStatusPaid
		go s.safeNotifyWall(payment, data)
		return nil
	case "failed", "TRADE_CLOSED":
		if err := s.repo.UpdateStatus(paymentID, model.PaymentStatusFailed, data.ChannelTxID); err != nil {
			return err
		}
		payment.Status = model.PaymentStatusFailed
		go s.safeNotifyWall(payment, data)
		return nil
	default:
		return nil
	}
}

func (s *PaymentService) safeNotifyWall(payment *model.Payment, data *channel.CallbackData) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[pay] panic in notifyWall: %v", r)
		}
	}()
	s.notifyWall(payment, data)
}

func (s *PaymentService) notifyWall(payment *model.Payment, data *channel.CallbackData) {
	if s.wallWebhookURL == "" {
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
		log.Printf("[pay] failed to marshal webhook payload: %v", err)
		return
	}

	retries := []time.Duration{1 * time.Second, 5 * time.Second, 15 * time.Second}
	var lastErr error
	for i, delay := range retries {
		if i > 0 {
			time.Sleep(delay)
		}
		if err := s.sendWebhook(body); err != nil {
			lastErr = err
			log.Printf("[pay] webhook attempt %d/3 failed: %v", i+1, err)
			continue
		}
		return
	}
	log.Printf("[pay] webhook permanently failed after 3 attempts: %v", lastErr)
}

func (s *PaymentService) sendWebhook(body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.wallWebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

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

func marshalOrEmpty(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
