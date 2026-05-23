package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// App represents an application that uses the payment API.
type App struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"type:varchar(255);not null"`
	APIKey    string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	Status    string    `gorm:"type:varchar(20);default:active"`

	// Service provider sub-merchant fields
	AlipayPID       string `gorm:"type:varchar(64)"`
	WechatSubMchid  string `gorm:"type:varchar(32)"`
	WechatSubAppid  string `gorm:"type:varchar(32)"`

	// Callback URL for notifying this app when payment status changes
	WebhookURL string `gorm:"type:varchar(500)"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// GenerateAPIKey creates a random API key with prefix.
func GenerateAPIKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "sk_" + hex.EncodeToString(b)
}