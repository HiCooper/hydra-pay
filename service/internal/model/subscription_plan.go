package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	PlanStatusActive   = "active"
	PlanStatusArchived = "archived"
)

// SubscriptionPlan defines a recurring billing plan (e.g., "Pro Monthly").
type SubscriptionPlan struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Amount      int64     `gorm:"not null"`
	Currency    string    `gorm:"type:varchar(10);not null;default:CNY"`
	Interval    string    `gorm:"type:varchar(20);not null"`
	Description string    `gorm:"type:varchar(500)"`
	Status      string    `gorm:"type:varchar(20);default:active"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
