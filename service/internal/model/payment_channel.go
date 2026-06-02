package model

import (
	"time"

	"github.com/google/uuid"
)

// PaymentChannel represents a payment channel integrated in hydra-pay.
// This is the single source of truth for channel metadata — which channels
// exist, their display labels, and whether they support merchant self-service
// onboarding.  Adapter instantiation and per-channel callback schemas remain
// code-level concerns.
type PaymentChannel struct {
	ID                 uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Key                string    `gorm:"type:varchar(32);uniqueIndex;not null"         json:"key"`
	Code               string    `gorm:"type:varchar(8);not null"                         json:"code"`
	Label              string    `gorm:"type:varchar(64);not null"                     json:"label"`
	SupportsOnboarding bool      `gorm:"default:false"                                 json:"supports_onboarding"`
	Enabled            bool      `gorm:"default:true"                                  json:"enabled"`
	SortOrder          int       `gorm:"default:0"                                     json:"sort_order"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
