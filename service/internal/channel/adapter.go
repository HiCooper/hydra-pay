package channel

import (
	"context"

	"github.com/hydra/pay-service/internal/model"
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
// Contains the normalized fields plus the channel-specific callback record ready for persistence.
type CallbackResult struct {
	ChannelTxID string
	PaymentID   string
	Status      string
	Amount      int64
	Currency    string

	// Channel-specific callback records — set by the respective adapter.
	AlipayCallback *model.AlipayCallback
	WechatPayCallback *model.WechatPayCallback
}

// RefundRequest is the unified request to create a refund on any channel.
type RefundRequest struct {
	TradeNo       string // hydra-pay trade_no
	ChannelTxID   string // channel transaction ID (for channels that require it)
	RefundAmount  int64  // refund amount in cents
	TotalAmount   int64  // original payment amount in cents
	RefundReason  string
	OutRequestNo  string // deduplication key
	SubMerchantID string // service provider mode: sub merchant ID
}

// RefundResponse is the unified response from creating a refund.
type RefundResponse struct {
	ChannelRefundID string
	RefundFee       int64 // actual refund amount in cents
	RawResponse     map[string]interface{}
}

// ---- Onboarding types ----

// OnboardingRequest is the unified request to initiate merchant onboarding.
type OnboardingRequest struct {
	OutRequestNo string
	MerchantName string
	ContactName  string
	ContactPhone string
	ContactEmail string
	NotifyURL    string
}

// OnboardingResponse is the unified response from initiating onboarding.
type OnboardingResponse struct {
	ApplymentID string
	SignURL     string
	QRCodeURL   string
	Status      string
	RawResponse map[string]interface{}
}

// OnboardingStatusResponse is returned when querying onboarding status.
type OnboardingStatusResponse struct {
	ApplymentID   string
	Status        string
	SubMerchantID string
	SignURL       string
	QRCodeURL     string
	RawResponse   map[string]interface{}
}

// OnboardingCallbackResult is the verified, parsed onboarding callback.
type OnboardingCallbackResult struct {
	ApplymentID   string
	OutRequestNo  string
	Status        string
	SubMerchantID string
	RejectReason  string
	RawBody       string
}

// OnboardingProvider is an optional interface for channels that support
// merchant self-service onboarding (间联商户进件).
type OnboardingProvider interface {
	SubmitOnboarding(ctx context.Context, req *OnboardingRequest) (*OnboardingResponse, error)
	QueryOnboarding(ctx context.Context, applymentID string) (*OnboardingStatusResponse, error)
	VerifyOnboardingCallback(ctx context.Context, data *CallbackData) (*OnboardingCallbackResult, error)
}

// Adapter is the interface that every payment channel must implement.
type Adapter interface {
	Name() string
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error)
	VerifyCallback(ctx context.Context, data *CallbackData) (*CallbackResult, error)
	GetPaymentStatus(ctx context.Context, channelTxID string) (string, error)
	Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)
}