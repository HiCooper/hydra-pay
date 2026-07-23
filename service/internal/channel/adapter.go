package channel

import (
	"context"

	"github.com/hydra/pay-service/internal/model"
)

// CreatePaymentRequest is the unified request to create a payment on any channel.
type CreatePaymentRequest struct {
	PaymentID    string
	Amount       int64
	Currency     string
	Description  string
	SuccessURL   string
	CancelURL    string
	TradeType    string // "native", "app", "jsapi", "h5", "miniapp"
	OpenID       string // WeChat JSAPI/miniapp user openid
	ChannelAppID string // merchant appid for the channel
	ClientIP     string // client IP for WeChat JSAPI
	NotifyURL    string // override the default notify_url (for testing or per-payment callback)
}

// CreatePaymentResponse is the unified response from creating a payment.
type CreatePaymentResponse struct {
	ChannelTxID string
	PaymentURL  string
	QRCodeURL   string
	RawResponse map[string]interface{}
}

// CallbackData is the raw callback/notification data passed to the adapter for verification.
type CallbackData struct {
	RawBody []byte            // raw request body (form-encoded for Alipay, JSON for WeChat)
	Headers map[string]string // HTTP headers (WeChat V3 signature headers)
}

// CallbackResult is the verified, parsed result returned from the adapter.
// Contains the normalized fields plus the channel-specific callback record ready for persistence.
type CallbackResult struct {
	ChannelTxID string
	PaymentID   string
	Status      string
	Amount      int64
	Currency    string

	// Channel-specific callback records — set by the respective adapter.
	AlipayCallback    *model.AlipayCallback
	WechatPayCallback *model.WechatPayCallback
	UnionpayCallback  *model.UnionpayCallback
	EcnyCallback      *model.EcnyCallback
}

// RefundRequest is the unified request to create a refund on any channel.
type RefundRequest struct {
	TradeNo      string // hydra-pay trade_no
	ChannelTxID  string // channel transaction ID (for channels that require it)
	RefundAmount int64  // refund amount in cents
	TotalAmount  int64  // original payment amount in cents
	RefundReason string
	OutRequestNo string // deduplication key
}

// RefundResponse is the unified response from creating a refund.
type RefundResponse struct {
	ChannelRefundID string
	RefundFee       int64 // actual refund amount in cents
	RawResponse     map[string]interface{}
}


// Adapter is the interface that every payment channel must implement.
type Adapter interface {
	Name() string
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error)
	VerifyCallback(ctx context.Context, data *CallbackData) (*CallbackResult, error)
	GetPaymentStatus(ctx context.Context, channelTxID string) (string, error)
	Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)
}