package repository

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Record(event *model.PaymentEvent) error {
	return r.db.Create(event).Error
}

func (r *EventRepository) ListByPayment(paymentID uuid.UUID) ([]model.PaymentEvent, error) {
	var events []model.PaymentEvent
	err := r.db.Where("payment_id = ?", paymentID).
		Order("created_at ASC").
		Find(&events).Error
	return events, err
}

// RecordEvent is a thin helper that constructs and saves a PaymentEvent.
// Designed for use from the service layer where we have the full context.
func RecordEvent(db *gorm.DB, eventType, channel string, paymentID uuid.UUID, rawBody string, result map[string]interface{}, errMsg string) {
	event := &model.PaymentEvent{
		PaymentID: paymentID,
		Type:      eventType,
		Channel:   channel,
		RawBody:   rawBody,
		Error:     errMsg,
	}
	if result != nil {
		b, _ := json.Marshal(result)
		event.Result = datatypes.JSON(b)
	}
	db.Create(event)
}
