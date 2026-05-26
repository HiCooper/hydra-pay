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
	UserID             string     `gorm:"type:varchar(255);not null;index"`
	PlanID             string     `gorm:"type:varchar(255);not null;index"`
	Status             string     `gorm:"type:varchar(20);default:active;index:idx_sub_period,priority:1"`
	CurrentPeriodStart time.Time  `gorm:"not null"`
	CurrentPeriodEnd   time.Time  `gorm:"not null;index:idx_sub_period,priority:2"`
	CancelledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
