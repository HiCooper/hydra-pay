package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// WeChatCallback stores every field from a WeChat Pay V3 callback notification
// after AES-GCM decryption. Notification ID is unique for idempotency.
type WeChatCallback struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PaymentID uuid.UUID `gorm:"type:uuid;not null;index"`

	// --- Notification identity ---
	NotificationID string `gorm:"type:varchar(64);uniqueIndex"` // WeChat notification id
	EventType      string `gorm:"type:varchar(64)"`               // TRANSACTION.SUCCESS

	// --- Transaction ---
	TransactionID string `gorm:"type:varchar(64);index"` // 微信支付订单号
	OutTradeNo    string `gorm:"type:varchar(64);index"`  // 商户订单号 = 我们的 TradeNo
	TradeType     string `gorm:"type:varchar(16)"`         // JSAPI, NATIVE, APP, MICROPAY
	TradeState    string `gorm:"type:varchar(32)"`          // SUCCESS, REFUND, NOTPAY, CLOSED ...
	TradeStateDesc string `gorm:"type:varchar(256)"`        // 交易状态描述

	// --- Amount ---
	AmountTotal        int64  `gorm:""`                   // 订单金额（分）
	AmountPayerTotal   int64  `gorm:""`                   // 用户支付金额（分）
	AmountCurrency     string `gorm:"type:varchar(10)"`
	AmountPayerCurrency string `gorm:"type:varchar(10)"`

	// --- Payer ---
	PayerOpenid string `gorm:"type:varchar(64);index"`

	// --- Merchant ---
	Mchid  string `gorm:"type:varchar(32)"`
	Appid  string `gorm:"type:varchar(32)"`
	Attach string `gorm:"type:varchar(256)"`

	// --- Service provider ---
	SpAppid    string `gorm:"type:varchar(32)"`
	SpMchid    string `gorm:"type:varchar(32);index"`
	SubAppid   string `gorm:"type:varchar(32)"`
	SubMchid   string `gorm:"type:varchar(32);index"`

	// --- Other ---
	BankType      string `gorm:"type:varchar(16)"`
	SuccessTime   string `gorm:"type:varchar(32)"`
	PromotionDetail datatypes.JSON `gorm:"type:jsonb"`

	// --- Raw ---
	RawBody string `gorm:"type:text"`

	CreatedAt time.Time
}