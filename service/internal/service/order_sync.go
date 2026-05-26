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

		// Already in terminal state — no action needed
		if payment.Status != model.PaymentStatusProcessing {
			taskRepo.MarkDone(task.ID)
			continue
		}

		adapter, err := getAdapter(payment.Channel)
		if err != nil {
			logger.Error(ctx, "skip task", "trade_no", payment.TradeNo, "error", err)
			continue
		}

		queryID := payment.ExternalID
		if queryID == "" {
			queryID = payment.TradeNo
		}

		status, err := adapter.GetPaymentStatus(ctx, queryID)
		if err != nil {
			logger.Error(ctx, "query failed", "trade_no", payment.TradeNo, "error", err)
					taskRepo.MarkDone(task.ID)
			continue
		}

		switch status {
		case model.PaymentStatusFailed:
			if err := paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusFailed, payment.ExternalID); err != nil {
				logger.Error(ctx, "update to failed", "trade_no", payment.TradeNo, "error", err)
			}
			repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
				payment.ID, "",
				map[string]interface{}{"from": model.PaymentStatusProcessing, "to": model.PaymentStatusFailed, "source": "sync"}, "")
			logger.Info(ctx, "order closed → failed", "trade_no", payment.TradeNo)
			taskRepo.MarkDone(task.ID)

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
			taskRepo.MarkDone(task.ID)

		default:
			// still pending — leave task as-is for next tick
		}
	}
}