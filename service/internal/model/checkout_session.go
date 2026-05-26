package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	CheckoutSessionOpen      = "open"
	CheckoutSessionCompleted = "completed"
	CheckoutSessionExpired   = "expired"
)

// CheckoutSession represents a Stripe-style checkout session.
// Created by the merchant, consumed by the hosted checkout page.
type CheckoutSession struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AppID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	Amount      int64          `gorm:"not null"`
	Currency    string         `gorm:"type:varchar(10);not null;default:CNY"`
	Description string         `gorm:"type:varchar(500)"`
	SuccessURL  string         `gorm:"type:varchar(500)"`
	CancelURL   string         `gorm:"type:varchar(500)"`
	Metadata    datatypes.JSON `gorm:"type:jsonb"`
	Status      string         `gorm:"type:varchar(20);not null;default:open"`
	PaymentID   *uuid.UUID     `gorm:"type:uuid;index"` // set after user activates
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
