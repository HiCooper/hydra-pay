package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AlipayCallback stores every parameter from an Alipay async notification (notify).
// Unique on notify_id for idempotency.
type AlipayCallback struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PaymentID uuid.UUID `gorm:"type:uuid;not null;index"`

	// --- Notification identity ---
	NotifyID   string `gorm:"type:varchar(128);uniqueIndex"`
	NotifyType string `gorm:"type:varchar(64)"`
	NotifyTime string `gorm:"type:varchar(32)"`
	SignType   string `gorm:"type:varchar(16)"`

	// --- Trade ---
	TradeNo      string `gorm:"type:varchar(64);index"` // 支付宝交易号
	OutTradeNo   string `gorm:"type:varchar(64);index"`  // 商户订单号 = 我们的 TradeNo
	TradeStatus  string `gorm:"type:varchar(32)"`         // TRADE_SUCCESS, WAIT_BUYER_PAY, TRADE_CLOSED ...
	Subject      string `gorm:"type:varchar(256)"`
	TotalAmount  string `gorm:"type:varchar(16)"`  // 支付宝金额为字符串（元）
	ReceiptAmount string `gorm:"type:varchar(16)"`
	BuyerPayAmount string `gorm:"type:varchar(16)"`
	PointAmount  string `gorm:"type:varchar(16)"`
	InvoiceAmount string `gorm:"type:varchar(16)"`

	// --- Payer ---
	BuyerID      string `gorm:"type:varchar(64);index"`
	BuyerLogonID string `gorm:"type:varchar(128)"`

	// --- Time ---
	GmtCreate   string `gorm:"type:varchar(32)"`
	GmtPayment  string `gorm:"type:varchar(32)"`
	GmtClose    string `gorm:"type:varchar(32)"`

	// --- Complex / nested ---
	FundBillList      datatypes.JSON `gorm:"type:jsonb"` // [{"amount":"...","fundChannel":"ALIPAYACCOUNT"}]
	VoucherDetailList datatypes.JSON `gorm:"type:jsonb"`
	PassbackParams    string         `gorm:"type:varchar(512)"`

	// --- Raw ---
	RawBody string `gorm:"type:text"`

	CreatedAt time.Time
}