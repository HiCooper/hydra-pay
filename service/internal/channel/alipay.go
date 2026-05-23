package channel

import (
	"context"
	"fmt"

	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/errors"
)

// AlipayAdapter implements the Adapter interface for Alipay.
// MVP: uses a simulated implementation. Replace with real Alipay SDK calls
// when merchant credentials are available.
type AlipayAdapter struct{}

func NewAlipayAdapter() *AlipayAdapter {
	return &AlipayAdapter{}
}

func (a *AlipayAdapter) Name() string { return model.ChannelAlipay }

func (a *AlipayAdapter) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	if req.Amount <= 0 {
		return nil, errors.New(errors.ValidationError, "amount must be positive")
	}

	// In production, this would call Alipay's unified order API:
	//   alipay.trade.precreate (QR code) or alipay.trade.app.pay (App payment)
	// The response would include a payment URL or QR code URL.

	channelTxID := fmt.Sprintf("alipay_%s", req.PaymentID)

	return &CreatePaymentResponse{
		ChannelTxID: channelTxID,
		PaymentURL:  fmt.Sprintf("https://openapi.alipay.com/gateway.do?out_trade_no=%s&total_amount=%.2f", req.PaymentID, float64(req.Amount)/100.0),
		QRCodeURL:   fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?data=alipay://pay/%s", req.PaymentID),
		RawResponse: map[string]interface{}{
			"code":    "10000",
			"msg":     "Success",
			"out_trade_no": req.PaymentID,
		},
	}, nil
}

func (a *AlipayAdapter) VerifyCallback(ctx context.Context, data *CallbackData) error {
	// In production: verify the Alipay signature using the public key.
	// Alipay signs notifications with its private key; verify with Alipay's public key.
	// Also call alipay.trade.query to double-check the payment status.

	if data.ChannelTxID == "" {
		return errors.New(errors.InvalidSignature, "missing channel transaction ID")
	}

	// Stub: accept all callbacks where channelTxID is present.
	// TODO: Implement real RSA signature verification with Alipay public key.
	return nil
}

func (a *AlipayAdapter) GetPaymentStatus(ctx context.Context, channelTxID string) (string, error) {
	// In production: call alipay.trade.query
	// Stub: always return paid
	return model.PaymentStatusPaid, nil
}
