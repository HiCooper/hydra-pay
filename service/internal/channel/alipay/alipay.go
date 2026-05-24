package alipay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/smartwalle/alipay/v3"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/errors"
)

// Adapter implements the channel.Adapter interface for Alipay.
type Adapter struct {
	client *alipay.Client
	config *config.AlipayConfig
}

// NewAdapter creates a production Alipay adapter with real SDK integration.
func NewAdapter(cfg *config.AlipayConfig) (*Adapter, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("alipay: ALIPAY_APP_ID is required")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("alipay: private key is required (set ALIPAY_PRIVATE_KEY or ALIPAY_PRIVATE_KEY_PATH)")
	}

	// SDK New() takes a "production" bool: true=prod gateway, false=sandbox gateway
	client, err := alipay.New(cfg.AppID, cfg.PrivateKey, !cfg.IsSandbox)
	if err != nil {
		return nil, fmt.Errorf("alipay: failed to create client: %w", err)
	}

	if cfg.AlipayPublicKey != "" {
		if err := client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
			return nil, fmt.Errorf("alipay: failed to load Alipay public key: %w", err)
		}
		log.Println("[alipay] public key loaded successfully")
	} else {
		log.Println("[alipay] WARNING: no public key configured — callback verification will fail")
	}

	log.Printf("[alipay] adapter initialized (app_id=%s, sandbox=%v)", cfg.AppID, cfg.IsSandbox)
	return &Adapter{client: client, config: cfg}, nil
}

func (a *Adapter) Name() string { return model.ChannelAlipay }

func (a *Adapter) CreatePayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	if req.Amount <= 0 {
		return nil, errors.New(errors.ValidationError, "amount must be positive")
	}
	if req.TradeType == "" {
		req.TradeType = "native"
	}

	amountYuan := formatAmount(req.Amount)

	switch req.TradeType {
	case "native":
		return a.createQRCodePayment(ctx, req, amountYuan)
	case "app":
		return a.createAppPayment(ctx, req, amountYuan)
	case "jsapi", "h5":
		return a.createH5Payment(ctx, req, amountYuan)
	default:
		return nil, errors.New(errors.ValidationError, "unsupported alipay trade type: "+req.TradeType)
	}
}

func (a *Adapter) createQRCodePayment(ctx context.Context, req *channel.CreatePaymentRequest, amount string) (*channel.CreatePaymentResponse, error) {
	p := alipay.TradePreCreate{}
	p.OutTradeNo = req.PaymentID
	p.TotalAmount = amount
	p.Subject = truncate(req.Description, 256)
	p.ProductCode = "FACE_TO_FACE_PAYMENT"
	p.NotifyURL = a.config.NotifyURL
	if req.NotifyURL != "" {
		p.NotifyURL = req.NotifyURL
	}
	if req.SubMerchantID != "" {
		p.SellerId = req.SubMerchantID
	}

	resp, err := a.client.TradePreCreate(ctx, p)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "alipay precreate failed", err)
	}
	if resp.Code != alipay.CodeSuccess {
		return nil, errors.New(errors.ChannelError,
			fmt.Sprintf("alipay precreate error: %s (code=%s, sub_code=%s)", resp.Msg, resp.Code, resp.SubCode))
	}

	log.Printf("[alipay] precreate success: out_trade_no=%s, qr_code=%s", req.PaymentID, resp.QRCode)

	return &channel.CreatePaymentResponse{
		ChannelTxID: "",
		QRCodeURL:   resp.QRCode,
		RawResponse: structToMap(resp),
	}, nil
}

func (a *Adapter) createAppPayment(ctx context.Context, req *channel.CreatePaymentRequest, amount string) (*channel.CreatePaymentResponse, error) {
	p := alipay.TradeAppPay{}
	p.OutTradeNo = req.PaymentID
	p.TotalAmount = amount
	p.Subject = truncate(req.Description, 256)
	p.ProductCode = "QUICK_MSECURITY_PAY"
	p.NotifyURL = a.config.NotifyURL
	if req.NotifyURL != "" {
		p.NotifyURL = req.NotifyURL
	}
	if req.SubMerchantID != "" {
		p.SellerId = req.SubMerchantID
	}

	orderStr, err := a.client.TradeAppPay(p)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "alipay app pay failed", err)
	}

	log.Printf("[alipay] app pay success: out_trade_no=%s", req.PaymentID)

	return &channel.CreatePaymentResponse{
		ChannelTxID: "",
		PaymentURL:  orderStr,
		RawResponse: map[string]interface{}{"order_string": orderStr},
	}, nil
}

func (a *Adapter) createH5Payment(ctx context.Context, req *channel.CreatePaymentRequest, amount string) (*channel.CreatePaymentResponse, error) {
	p := alipay.TradeWapPay{}
	p.OutTradeNo = req.PaymentID
	p.TotalAmount = amount
	p.Subject = truncate(req.Description, 256)
	p.ProductCode = "QUICK_WAP_WAY"
	p.NotifyURL = a.config.NotifyURL
	if req.NotifyURL != "" {
		p.NotifyURL = req.NotifyURL
	}
	p.ReturnURL = a.config.ReturnURL
	p.QuitURL = req.CancelURL
	if req.SubMerchantID != "" {
		p.SellerId = req.SubMerchantID
	}

	paymentURL, err := a.client.TradeWapPay(p)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "alipay wap pay failed", err)
	}

	log.Printf("[alipay] wap pay success: out_trade_no=%s", req.PaymentID)

	return &channel.CreatePaymentResponse{
		ChannelTxID: "",
		PaymentURL:  paymentURL.String(),
		RawResponse: map[string]interface{}{"payment_url": paymentURL.String()},
	}, nil
}

func (a *Adapter) GetPaymentStatus(ctx context.Context, channelTxID string) (string, error) {
	p := alipay.TradeQuery{}
	p.OutTradeNo = channelTxID

	resp, err := a.client.TradeQuery(ctx, p)
	if err != nil {
		return "", errors.Wrap(errors.ChannelError, "alipay query failed", err)
	}
	if resp.Code != alipay.CodeSuccess {
		return "", errors.New(errors.ChannelError,
			fmt.Sprintf("alipay query error: %s (code=%s)", resp.Msg, resp.Code))
	}

	return mapAlipayTradeStatus(string(resp.TradeStatus)), nil
}

func (a *Adapter) VerifyCallback(ctx context.Context, data *channel.CallbackData) (*channel.CallbackResult, error) {
	values, err := url.ParseQuery(string(data.RawBody))
	if err != nil {
		return nil, errors.New(errors.InvalidSignature, "failed to parse alipay callback body")
	}

	if err := a.client.VerifySign(ctx, values); err != nil {
		return nil, errors.Wrap(errors.InvalidSignature, "alipay signature verification failed", err)
	}

	outTradeNo := values.Get("out_trade_no")
	tradeNo := values.Get("trade_no")
	totalAmount := values.Get("total_amount")
	tradeStatus := values.Get("trade_status")
	notifyID := values.Get("notify_id")

	if outTradeNo == "" {
		return nil, errors.New(errors.ValidationError, "missing out_trade_no in alipay callback")
	}
	if tradeNo == "" {
		return nil, errors.New(errors.ValidationError, "missing trade_no in alipay callback")
	}

	log.Printf("[alipay] callback verified: out_trade_no=%s, trade_no=%s, notify_id=%s, status=%s",
		outTradeNo, tradeNo, notifyID, tradeStatus)

	// Double-check by querying Alipay API (anti-replay)
	status, err := a.GetPaymentStatus(ctx, outTradeNo)
	if err != nil {
		log.Printf("[alipay] callback double-check query failed: %v", err)
		status = mapAlipayTradeStatus(tradeStatus)
	}

	amountYuan, _ := strconv.ParseFloat(totalAmount, 64)
	amountCents := int64(amountYuan * 100)

	// Build full callback record with all Alipay parameters
	cb := &model.AlipayCallback{
		NotifyID:        notifyID,
		NotifyType:      values.Get("notify_type"),
		NotifyTime:      values.Get("notify_time"),
		SignType:        values.Get("sign_type"),
		TradeNo:         tradeNo,
		OutTradeNo:      outTradeNo,
		TradeStatus:     tradeStatus,
		Subject:         values.Get("subject"),
		TotalAmount:     totalAmount,
		ReceiptAmount:   values.Get("receipt_amount"),
		BuyerPayAmount:  values.Get("buyer_pay_amount"),
		PointAmount:     values.Get("point_amount"),
		InvoiceAmount:   values.Get("invoice_amount"),
		BuyerID:         values.Get("buyer_id"),
		BuyerLogonID:    values.Get("buyer_logon_id"),
		GmtCreate:       values.Get("gmt_create"),
		GmtPayment:      values.Get("gmt_payment"),
		GmtClose:        values.Get("gmt_close"),
		PassbackParams:  values.Get("passback_params"),
		FundBillList:    parseJSONField(values.Get("fund_bill_list")),
		VoucherDetailList: parseJSONField(values.Get("voucher_detail_list")),
		RawBody:         string(data.RawBody),
	}

	return &channel.CallbackResult{
		ChannelTxID:    tradeNo,
		PaymentID:      outTradeNo,
		Status:         status,
		Amount:         amountCents,
		Currency:       "CNY",
		AlipayCallback: cb,
	}, nil
}

func parseJSONField(s string) []byte {
	if s == "" {
		return nil
	}
	return []byte(s)
}

func mapAlipayTradeStatus(status string) string {
	switch status {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return model.PaymentStatusPaid
	case "WAIT_BUYER_PAY":
		return model.PaymentStatusPending
	case "TRADE_CLOSED":
		return model.PaymentStatusFailed
	default:
		return model.PaymentStatusPending
	}
}

func formatAmount(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100.0)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

func structToMap(v interface{}) map[string]interface{} {
	data, _ := json.Marshal(v)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}