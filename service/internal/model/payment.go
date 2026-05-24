package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	PaymentStatusPending    = "pending"
	PaymentStatusProcessing = "processing"
	PaymentStatusPaid       = "paid"
	PaymentStatusFailed     = "failed"
	PaymentStatusCancelled  = "cancelled"
	PaymentStatusRefunded   = "refunded"

	ChannelAlipay = "alipay"
	ChannelWechat = "wechat"
	ChannelStripe = "stripe"
)

// Payment represents a payment order.
type Payment struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TradeNo     string         `gorm:"type:varchar(32);not null;uniqueIndex"`
	AppID       uuid.UUID      `gorm:"type:uuid;not null;index"`
	UserID      string         `gorm:"type:varchar(255);not null"`
	PlanID      string         `gorm:"type:varchar(255)"`
	Amount      int64          `gorm:"not null"`
	Currency    string         `gorm:"type:varchar(10);not null;default:CNY"`
	Channel     string         `gorm:"type:varchar(50);not null"`
	Status      string         `gorm:"type:varchar(20);not null;default:pending"`
	ExternalID  string         `gorm:"type:varchar(255)"`
	Description string         `gorm:"type:varchar(500)"`
	SuccessURL  string         `gorm:"type:varchar(500)"`
	CancelURL   string         `gorm:"type:varchar(500)"`
	Metadata    datatypes.JSON `gorm:"type:jsonb"`
	PaidAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
