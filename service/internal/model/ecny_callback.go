package model

import (
	"time"

	"github.com/google/uuid"
)

// EcnyCallback stores every field from an e-CNY (数字人民币) service agency
// async notification callback. Unique on ChannelTxID for idempotency.
// Follows the same pattern as UnionpayCallback.
type EcnyCallback struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PaymentID uuid.UUID `gorm:"type:uuid;not null;index"`

	// --- Notification identity ---
	ChannelTxID string `gorm:"type:varchar(64);uniqueIndex"` // 受理机构交易流水号，唯一

	// --- Transaction ---
	OrderID  string `gorm:"type:varchar(64);index"` // 商户订单号 = 我们的 TradeNo
	TxnAmt   int64  `gorm:""`                        // 交易金额（分）
	TxnTime  string `gorm:"type:varchar(32)"`        // 交易时间
	RespCode string `gorm:"type:varchar(16)"`        // 响应码
	RespMsg  string `gorm:"type:varchar(256)"`       // 响应信息
	Status   string `gorm:"type:varchar(32)"`        // 交易状态: SUCCESS, PROCESSING, CLOSED, FAILED

	// --- Signature ---
	Signature  string `gorm:"type:varchar(512)"` // SM2 签名（Base64）
	SignMethod string `gorm:"type:varchar(8)"`   // 签名方法: "SM2"

	// --- Raw ---
	RawBody string `gorm:"type:text"`

	CreatedAt time.Time
}
