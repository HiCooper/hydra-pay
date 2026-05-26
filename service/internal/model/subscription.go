package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	SubscriptionStatusActive    = "active"
	SubscriptionStatusPastDue   = "past_due"
	SubscriptionStatusCancelled = "cancelled"
	SubscriptionStatusExpired   = "expired"
)

// Subscription represents a user's subscription to a plan.
type Subscription struct {
	ID                 uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AppID              uuid.UUID  `gorm:"type:uuid;not null;index"`
	UserID             string     `gorm:"type:varchar(128);not null;index"`
	PlanID             uuid.UUID  `gorm:"type:uuid;not null"`
	Status             string     `gorm:"type:varchar(20);default:active"`
	CurrentPeriodStart time.Time  `gorm:"not null"`
	CurrentPeriodEnd   time.Time  `gorm:"not null"`
	CancelledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
