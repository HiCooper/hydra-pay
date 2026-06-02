package service

import (
	"context"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/pkg/logger"
)

// SyncExpiredOrders fetches due scheduled tasks, queries the channel for real status, and updates.
func SyncExpiredOrders(db *gorm.DB, getAdapter func(string) (channel.Adapter, error)) {
	ctx := context.Background()
	taskRepo := repository.NewScheduledTaskRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	tasks, err := taskRepo.FetchDue(50)
	if err != nil {
		logger.Error(ctx, "failed to fetch due tasks", "error", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	logger.Info(ctx, "found due tasks", "count", len(tasks))

	for _, task := range tasks {
		payment, err := paymentRepo.GetByID(task.ReferenceID)
		if err != nil {
			taskRepo.MarkDone(task.ID)
			continue
		}

		// Non-terminal states only: pending & processing
		if payment.Status != model.PaymentStatusPending && payment.Status != model.PaymentStatusProcessing {
			taskRepo.MarkDone(task.ID)
			continue
		}

		// Payment never activated — mark as expired
		if payment.Status == model.PaymentStatusPending {
			logger.Info(ctx, "pending payment timed out", "trade_no", payment.TradeNo)
			paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusExpired, "")
			repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
				payment.ID, "",
				map[string]interface{}{"from": model.PaymentStatusPending, "to": model.PaymentStatusExpired, "source": "sync", "reason": "timeout"}, "")
			taskRepo.MarkDone(task.ID)
			continue
		}

		// Payment is processing — query channel for real status
		adapter, err := getAdapter(payment.Channel)
		if err != nil {
			// Channel not configured — expire the order
			logger.Info(ctx, "channel unavailable, expiring order", "trade_no", payment.TradeNo)
			paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusExpired, payment.ExternalID)
			repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
				payment.ID, "",
				map[string]interface{}{"from": model.PaymentStatusProcessing, "to": model.PaymentStatusExpired, "source": "sync", "reason": "channel_unavailable"}, "")
			taskRepo.MarkDone(task.ID)
			continue
		}

		queryID := payment.ExternalID
		if queryID == "" {
			queryID = payment.TradeNo
		}

		status, err := adapter.GetPaymentStatus(ctx, queryID)
		if err != nil {
			// Query error — expire the order
			logger.Info(ctx, "channel query failed, expiring order", "trade_no", payment.TradeNo, "error", err)
			paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusExpired, payment.ExternalID)
			repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
				payment.ID, "",
				map[string]interface{}{"from": model.PaymentStatusProcessing, "to": model.PaymentStatusExpired, "source": "sync", "reason": "query_error"}, "")
			taskRepo.MarkDone(task.ID)
			continue
		}

		switch status {
		case model.PaymentStatusFailed:
			paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusFailed, payment.ExternalID)
			repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
				payment.ID, "",
				map[string]interface{}{"from": model.PaymentStatusProcessing, "to": model.PaymentStatusFailed, "source": "sync"}, "")
			logger.Info(ctx, "order closed → failed", "trade_no", payment.TradeNo)

		case model.PaymentStatusPaid:
			applied, err := paymentRepo.MarkPaidIfPending(payment.ID, payment.ExternalID)
			if err != nil {
				logger.Error(ctx, "mark paid failed", "trade_no", payment.TradeNo, "error", err)
			}
			if applied {
				repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
					payment.ID, "",
					map[string]interface{}{"from": model.PaymentStatusProcessing, "to": model.PaymentStatusPaid, "source": "sync"}, "")
				logger.Info(ctx, "order paid (late catch)", "trade_no", payment.TradeNo)
			}

		default:
			// Still pending per channel beyond timeout — expire
			logger.Info(ctx, "order still pending after timeout, expiring", "trade_no", payment.TradeNo, "channel_status", status)
			paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusExpired, payment.ExternalID)
			repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
				payment.ID, "",
				map[string]interface{}{"from": model.PaymentStatusProcessing, "to": model.PaymentStatusExpired, "source": "sync", "reason": "timeout"}, "")
		}
		taskRepo.MarkDone(task.ID)
	}
}
