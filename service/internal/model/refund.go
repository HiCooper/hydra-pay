package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	RefundStatusSuccess    = "success"
	RefundStatusProcessing = "processing"
	RefundStatusFailed     = "failed"
)

// Refund stores the refund request and channel response for audit.
type Refund struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PaymentID uuid.UUID `gorm:"type:uuid;not null;index"`
	AppID     uuid.UUID `gorm:"type:uuid;not null;index"`
	TradeNo   string    `gorm:"type:varchar(32);index"` // hydra-pay trade_no

	// Request
	Channel        string `gorm:"type:varchar(32);not null"` // alipay / wechat
	RefundAmount   int64  `gorm:"not null"`                   // requested amount in cents
	RefundReason   string `gorm:"type:varchar(256)"`
	OutRequestNo   string `gorm:"type:varchar(64);uniqueIndex"` // 退费请求号，排重

	// Response
	Status          string         `gorm:"type:varchar(32)"`     // success / failed
	ChannelRefundID string         `gorm:"type:varchar(64)"`     // 支付宝 trade_no / 微信 refund_id
	ChannelTxID     string         `gorm:"type:varchar(64)"`     // 支付宝 trade_no / 微信 transaction_id
	RefundFee       int64          `gorm:"not null;default:0"`   // actual refunded amount in cents
	ResponseData    datatypes.JSON `gorm:"type:jsonb"`           // full channel response
	ErrorMsg        string         `gorm:"type:text"`            // error if failed

	CreatedAt time.Time
}