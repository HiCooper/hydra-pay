package channel

import (
	"context"
)

// CreatePaymentRequest is the unified request to create a payment on any channel.
type CreatePaymentRequest struct {
	PaymentID      string
	Amount         int64
	Currency       string
	Description    string
	SuccessURL     string
	CancelURL      string
	TradeType       string // "native", "app", "jsapi", "h5", "miniapp"
	OpenID          string // WeChat JSAPI/miniapp user openid
	ChannelAppID    string // WeChat direct: merchant appid. Service provider: sp_appid
	SubMerchantID   string // service provider mode: Alipay sub-merchant PID / WeChat sub_mchid
	SubChannelAppID string // WeChat service provider: sub-merchant's appid (sub_appid)
	ClientIP        string // client IP for WeChat JSAPI
	NotifyURL       string // override the default notify_url (for testing or per-payment callback)
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
type CallbackResult struct {
	ChannelTxID string
	PaymentID   string
	Status      string
	Amount      int64
	Currency    string
}

// Adapter is the interface that every payment channel must implement.
type Adapter interface {
	Name() string
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error)
	VerifyCallback(ctx context.Context, data *CallbackData) (*CallbackResult, error)
	GetPaymentStatus(ctx context.Context, channelTxID string) (string, error)
}