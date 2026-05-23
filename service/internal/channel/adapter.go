package channel

import (
	"context"
	"fmt"

	"github.com/hydra/pay-service/internal/model"
)

// CreatePaymentRequest is the unified request to create a payment on any channel.
type CreatePaymentRequest struct {
	PaymentID   string
	Amount      int64
	Currency    string
	Description string
	SuccessURL  string
	CancelURL   string
}

// CreatePaymentResponse is the unified response from creating a payment.
type CreatePaymentResponse struct {
	ChannelTxID string
	PaymentURL  string
	QRCodeURL   string
	RawResponse map[string]interface{}
}

// CallbackData is the normalized callback/notification from a payment channel.
type CallbackData struct {
	ChannelTxID string
	PaymentID   string
	Status      string
	Amount      int64
	Currency    string
	RawBody     []byte
	Signature   string
}

// Adapter is the interface that every payment channel must implement.
type Adapter interface {
	Name() string
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error)
	VerifyCallback(ctx context.Context, data *CallbackData) error
	GetPaymentStatus(ctx context.Context, channelTxID string) (string, error)
	// Refund not in MVP scope
}

// GetAdapter returns the channel adapter for the given channel name.
func GetAdapter(name string) (Adapter, error) {
	switch name {
	case model.ChannelAlipay:
		return NewAlipayAdapter(), nil
	// Future channels: wechat, stripe, apple_iap, google_billing
	default:
		return nil, fmt.Errorf("unsupported channel: %s", name)
	}
}
