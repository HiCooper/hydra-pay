package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	EventCreated         = "created"
	EventChannelRequest  = "channel_request"
	EventCallbackReceived = "callback_received"
	EventStatusChanged   = "status_changed"
	EventWebhookSent     = "webhook_sent"
	EventRefund          = "refund"
)

// PaymentEvent records every key action in a payment's lifecycle.
// Append-only — each callback, status change, and webhook attempt is a separate row.
type PaymentEvent struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PaymentID uuid.UUID      `gorm:"type:uuid;not null;index"`
	Type      string         `gorm:"type:varchar(50);not null;index"`
	Channel   string         `gorm:"type:varchar(50)"`
	RawBody   string         `gorm:"type:text"`
	Result    datatypes.JSON `gorm:"type:jsonb"`
	Error     string         `gorm:"type:text"`
	CreatedAt time.Time
}
