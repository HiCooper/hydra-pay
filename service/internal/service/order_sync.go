package service

import (
	"context"
	"log"

	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
)

// SyncExpiredOrders fetches due scheduled tasks, queries the channel for real status, and updates.
func SyncExpiredOrders(db *gorm.DB, getAdapter func(string) (channel.Adapter, error)) {
	ctx := context.Background()
	taskRepo := repository.NewScheduledTaskRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	tasks, err := taskRepo.FetchDue(50)
	if err != nil {
		log.Printf("[sync] failed to fetch due tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	log.Printf("[sync] found %d due tasks", len(tasks))

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
			log.Printf("[sync] skip %s: %v", payment.TradeNo, err)
			continue
		}

		queryID := payment.ExternalID
		if queryID == "" {
			queryID = payment.TradeNo
		}

		status, err := adapter.GetPaymentStatus(ctx, queryID)
		if err != nil {
			log.Printf("[sync] query failed for %s: %v", payment.TradeNo, err)
					taskRepo.MarkDone(task.ID)
			continue
		}

		switch status {
		case model.PaymentStatusFailed:
			if err := paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusFailed, payment.ExternalID); err != nil {
				log.Printf("[sync] update to failed for %s: %v", payment.TradeNo, err)
			}
			repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
				payment.ID, "",
				map[string]interface{}{"from": model.PaymentStatusProcessing, "to": model.PaymentStatusFailed, "source": "sync"}, "")
			log.Printf("[sync] %s closed → failed", payment.TradeNo)
			taskRepo.MarkDone(task.ID)

		case model.PaymentStatusPaid:
			applied, err := paymentRepo.MarkPaidIfPending(payment.ID, payment.ExternalID)
			if err != nil {
				log.Printf("[sync] mark paid failed for %s: %v", payment.TradeNo, err)
			}
			if applied {
				repository.RecordEvent(db, model.EventStatusChanged, payment.Channel,
					payment.ID, "",
					map[string]interface{}{"from": model.PaymentStatusProcessing, "to": model.PaymentStatusPaid, "source": "sync"}, "")
				log.Printf("[sync] %s paid (late catch)", payment.TradeNo)
			}
			taskRepo.MarkDone(task.ID)

		default:
			// still pending — leave task as-is for next tick
		}
	}
}