package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// App represents an application that uses the payment API.
type App struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	MerchantID uuid.UUID `gorm:"type:uuid;not null;index"                       json:"merchant_id"`
	Name       string    `gorm:"type:varchar(255);not null"                      json:"name"`
	APIKey     string    `gorm:"type:varchar(255);not null;uniqueIndex"          json:"api_key"`
	Status     string    `gorm:"type:varchar(20);default:active"                 json:"status"`

	// Callback URL for notifying this app when payment status changes
	WebhookURL    string `gorm:"type:varchar(500)" json:"webhook_url"`
	WebhookSecret string `gorm:"type:varchar(255)" json:"webhook_secret"` // HMAC signing secret for webhook verification

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GenerateAPIKey creates a random API key with prefix.
func GenerateAPIKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "sk_" + hex.EncodeToString(b)
}