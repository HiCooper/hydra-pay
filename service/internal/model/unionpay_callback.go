package model

import (
	"time"

	"github.com/google/uuid"
)

// UnionpayCallback stores every field from a UnionPay (银联/云闪付) async notification.
// Unique on QueryID for idempotency.
type UnionpayCallback struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PaymentID uuid.UUID `gorm:"type:uuid;not null;index"`

	// --- Notification identity ---
	QueryID string `gorm:"type:varchar(64);uniqueIndex"` // 银联查询流水号，唯一

	// --- Transaction ---
	OrderID  string `gorm:"type:varchar(64);index"` // 商户订单号 = 我们的 TradeNo
	TxnTime  string `gorm:"type:varchar(32)"`         // 订单发送时间 (YYYYMMDDHHmmss)
	TxnAmt   int64  `gorm:""`                         // 交易金额（分）
	RespCode string `gorm:"type:varchar(8)"`           // 响应码 ("00" = 成功)
	RespMsg  string `gorm:"type:varchar(256)"`         // 响应信息

	// --- Settlement ---
	SettleAmt          int64  `gorm:""`                 // 清算金额（分）
	SettleCurrencyCode string `gorm:"type:varchar(3)"`  // 清算币种
	SettleDate         string `gorm:"type:varchar(8)"`  // 清算日期 (MMDD)

	// --- Trace ---
	TraceNo   string `gorm:"type:varchar(64)"`  // 系统跟踪号
	TraceTime string `gorm:"type:varchar(32)"`  // 交易传输时间

	// --- Additional ---
	ExchangeRate string `gorm:"type:varchar(16)"` // 汇率
	AccNo        string `gorm:"type:varchar(10)"` // 账号（掩码）
	PayCardType  string `gorm:"type:varchar(8)"`  // 支付卡类型

	// ���-- Signature ---
	Signature  string `gorm:"type:varchar(512)"` // 银联应答签名
	SignMethod string `gorm:"type:varchar(8)"`    // 签名方法: "01"=RSA, "12"=SM2

	// --- Raw ---
	RawBody string `gorm:"type:text"`

	CreatedAt time.Time
}
