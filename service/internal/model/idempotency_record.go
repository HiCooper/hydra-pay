package model

import (
	"time"

	"github.com/google/uuid"
)

// IdempotencyRecord stores cached API responses for idempotent request replay.
// Each record is scoped to an app + idempotency key pair. Only 2xx responses are cached.
type IdempotencyRecord struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	IdempotencyKey string    `gorm:"type:varchar(255);uniqueIndex:idx_app_idempotency;not null"`
	AppID          uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_app_idempotency"`
	ResponseStatus int       `gorm:"not null"`
	ResponseBody   string    `gorm:"type:text;not null"`
	CreatedAt      time.Time
	ExpiresAt      time.Time `gorm:"not null;index"`
}
