package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// App represents an application that uses the payment API.
type App struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MerchantID uuid.UUID `gorm:"type:uuid;not null;index"`
	Name       string    `gorm:"type:varchar(255);not null"`
	APIKey     string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	Status     string    `gorm:"type:varchar(20);default:active"`

	// Callback URL for notifying this app when payment status changes
	WebhookURL    string `gorm:"type:varchar(500)"`
	WebhookSecret string `gorm:"type:varchar(255)"` // HMAC signing secret for webhook verification

	CreatedAt time.Time
	UpdatedAt time.Time
}

// GenerateAPIKey creates a random API key with prefix.
func GenerateAPIKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "sk_" + hex.EncodeToString(b)
}