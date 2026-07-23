package wechat

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/app"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/logger"
	"github.com/hydra/pay-service/pkg/metrics"
)

// Adapter implements the channel.Adapter interface for WeChat Pay V3.
type Adapter struct {
	client          *core.Client
	mchID           string
	apiV3Key        string
	notifyURL       string
	nativeSvc       *native.NativeApiService
	jsapiSvc        *jsapi.JsapiApiService
	appSvc          *app.AppApiService
}

// NewAdapter creates a production WeChat Pay adapter with official SDK integration.
func NewAdapter(cfg *config.WechatConfig) (*Adapter, error) {
	if cfg.MchID == "" {
		return nil, fmt.Errorf("wechat: WECHAT_MCH_ID is required")
	}
	if cfg.APIv3Key == "" {
		return nil, fmt.Errorf("wechat: WECHAT_API_V3_KEY is required")
	}
	if cfg.SerialNo == "" {
		return nil, fmt.Errorf("wechat: WECHAT_SERIAL_NO is required")
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("wechat: private key is required (set WECHAT_PRIVATE_KEY or WECHAT_PRIVATE_KEY_PATH)")
	}

	privateKey, err := utils.LoadPrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to load private key: %w", err)
	}

	client, err := core.NewClient(
		context.Background(),
		option.WithWechatPayAutoAuthCipher(cfg.MchID, cfg.SerialNo, privateKey, cfg.APIv3Key),
	)
	if err != nil {
		return nil, fmt.Errorf("wechat: failed to create client: %w", err)
	}

	logger.Info(context.Background(), "adapter initialized", "mch_id", cfg.MchID, "serial", cfg.SerialNo)
	return &Adapter{
		client:           client,
		mchID:            cfg.MchID,
		apiV3Key:         cfg.APIv3Key,
		notifyURL:        cfg.NotifyURL,
		nativeSvc:        &native.NativeApiService{Client: client},
		jsapiSvc:         &jsapi.JsapiApiService{Client: client},
		appSvc:           &app.AppApiService{Client: client},
	}, nil
}

func (a *Adapter) Name() string { return model.ChannelWechat }

func (a *Adapter) CreatePayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	if req.Amount <= 0 {
		return nil, errors.New(errors.ValidationError, "amount must be positive")
	}
	if req.TradeType == "" {
		req.TradeType = "native"
	}

	return a.createDirectPayment(ctx, req)
}

func (a *Adapter) createDirectPayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	switch req.TradeType {
	case "native":
		return a.createNativePayment(ctx, req)
	case "jsapi", "miniapp":
		return a.createJSAPIPayment(ctx, req)
	case "app":
		return a.createAppPayment(ctx, req)
	default:
		return nil, errors.New(errors.ValidationError, "unsupported wechat trade type: "+req.TradeType)
	}
}


// --- Direct merchant payments ---

func (a *Adapter) createNativePayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	appid := req.ChannelAppID
	if appid == "" {
		return nil, errors.New(errors.ValidationError, "channel_app_id (WeChat AppID) is required for native payment")
	}

	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "wechat.native_prepay",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "wechat"), attribute.String("operation", "native_prepay")),
	)
	defer span.End()

	e, b := sentinel.Entry("wechat")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "wechat circuit breaker open")
	}
	start := time.Now()
	resp, _, err := a.nativeSvc.Prepay(ctx,
		native.PrepayRequest{
			Appid:       core.String(appid),
			Mchid:       core.String(a.mchID),
			Description: core.String(truncate(req.Description, 127)),
			OutTradeNo:  core.String(req.PaymentID),
			NotifyUrl:   core.String(notifyURL(req.NotifyURL, a.notifyURL)),
			Amount: &native.Amount{
				Currency: core.String(getCurrency(req.Currency)),
				Total:    core.Int64(req.Amount),
			},
		},
	)
	metrics.ChannelAPIRequestDuration.WithLabelValues("wechat", "native_prepay").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("wechat", "native_prepay", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "wechat native prepay failed", err)
	}
	e.Exit()

	logger.Info(ctx, "native prepay success", "out_trade_no", req.PaymentID)
	return &channel.CreatePaymentResponse{
		ChannelTxID: "",
		QRCodeURL:   *resp.CodeUrl,
		RawResponse: map[string]interface{}{"code_url": *resp.CodeUrl},
	}, nil
}

func (a *Adapter) createJSAPIPayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	appid := req.ChannelAppID
	if appid == "" {
		return nil, errors.New(errors.ValidationError, "channel_app_id is required for jsapi payment")
	}
	if req.OpenID == "" {
		return nil, errors.New(errors.ValidationError, "open_id is required for jsapi payment")
	}

	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "wechat.jsapi_prepay",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "wechat"), attribute.String("operation", "jsapi_prepay")),
	)
	defer span.End()

	e, b := sentinel.Entry("wechat")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "wechat circuit breaker open")
	}
	start := time.Now()
	resp, _, err := a.jsapiSvc.Prepay(ctx,
		jsapi.PrepayRequest{
			Appid:       core.String(appid),
			Mchid:       core.String(a.mchID),
			Description: core.String(truncate(req.Description, 127)),
			OutTradeNo:  core.String(req.PaymentID),
			NotifyUrl:   core.String(notifyURL(req.NotifyURL, a.notifyURL)),
			Amount: &jsapi.Amount{
				Currency: core.String(getCurrency(req.Currency)),
				Total:    core.Int64(req.Amount),
			},
			Payer: &jsapi.Payer{Openid: core.String(req.OpenID)},
		},
	)
	metrics.ChannelAPIRequestDuration.WithLabelValues("wechat", "jsapi_prepay").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("wechat", "jsapi_prepay", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "wechat jsapi prepay failed", err)
	}
	e.Exit()

	logger.Info(ctx, "jsapi prepay success", "out_trade_no", req.PaymentID, "prepay_id", *resp.PrepayId)
	return &channel.CreatePaymentResponse{
		ChannelTxID: "",
		PaymentURL:  *resp.PrepayId,
		RawResponse: structToMap(resp),
	}, nil
}

func (a *Adapter) createAppPayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	appid := req.ChannelAppID
	if appid == "" {
		return nil, errors.New(errors.ValidationError, "channel_app_id is required for app payment")
	}

	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "wechat.app_prepay",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "wechat"), attribute.String("operation", "app_prepay")),
	)
	defer span.End()

	e, b := sentinel.Entry("wechat")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "wechat circuit breaker open")
	}
	start := time.Now()
	resp, _, err := a.appSvc.Prepay(ctx,
		app.PrepayRequest{
			Appid:       core.String(appid),
			Mchid:       core.String(a.mchID),
			Description: core.String(truncate(req.Description, 127)),
			OutTradeNo:  core.String(req.PaymentID),
			NotifyUrl:   core.String(notifyURL(req.NotifyURL, a.notifyURL)),
			Amount:      &app.Amount{Currency: core.String(getCurrency(req.Currency)), Total: core.Int64(req.Amount)},
		},
	)
	metrics.ChannelAPIRequestDuration.WithLabelValues("wechat", "app_prepay").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("wechat", "app_prepay", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "wechat app prepay failed", err)
	}
	e.Exit()

	logger.Info(ctx, "app prepay success", "out_trade_no", req.PaymentID, "prepay_id", *resp.PrepayId)
	return &channel.CreatePaymentResponse{
		ChannelTxID: "",
		PaymentURL:  *resp.PrepayId,
		RawResponse: structToMap(resp),
	}, nil
}
// Refund creates a domestic refund via the WeChat Pay V3 refunds API.
func (a *Adapter) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	svc := refunddomestic.RefundsApiService{Client: a.client}
	createReq := refunddomestic.CreateRequest{
		OutTradeNo:  core.String(req.TradeNo),
		OutRefundNo: core.String(req.OutRequestNo),
		Amount: &refunddomestic.AmountReq{
			Refund:   core.Int64(req.RefundAmount),
			Total:    core.Int64(req.TotalAmount),
			Currency: core.String("CNY"),
		},
	}
	if req.RefundReason != "" {
		createReq.Reason = core.String(req.RefundReason)
	}

	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "wechat.refund",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "wechat"), attribute.String("operation", "refund")),
	)
	defer span.End()

	e, b := sentinel.Entry("wechat")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "wechat circuit breaker open")
	}
	start := time.Now()
	resp, _, err := svc.Create(ctx, createReq)
	metrics.ChannelAPIRequestDuration.WithLabelValues("wechat", "refund").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("wechat", "refund", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "wechat refund failed", err)
	}
	e.Exit()

	refundFee := int64(0)
	if resp.Amount != nil && resp.Amount.Refund != nil {
		refundFee = *resp.Amount.Refund
	}

	logger.Info(ctx, "refund success", "out_trade_no", req.TradeNo, "refund_id", *resp.RefundId, "status", string(*resp.Status))

	return &channel.RefundResponse{
		ChannelRefundID: *resp.RefundId,
		RefundFee:       refundFee,
		RawResponse:     structToMap(resp),
	}, nil
}

// GetPaymentStatus queries the order status by out_trade_no.
func (a *Adapter) GetPaymentStatus(ctx context.Context, channelTxID string) (string, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "wechat.query_order",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "wechat"), attribute.String("operation", "query_order")),
	)
	defer span.End()

	e, b := sentinel.Entry("wechat")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return "", errors.New(errors.ChannelError, "wechat circuit breaker open")
	}
	start := time.Now()
	resp, _, err := a.nativeSvc.QueryOrderByOutTradeNo(ctx,
		native.QueryOrderByOutTradeNoRequest{
			OutTradeNo: core.String(channelTxID),
			Mchid:      core.String(a.mchID),
		},
	)
	metrics.ChannelAPIRequestDuration.WithLabelValues("wechat", "query_order").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("wechat", "query_order", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return "", errors.Wrap(errors.ChannelError, "wechat query order failed", err)
	}
	e.Exit()

	return mapWechatTradeState(*resp.TradeState), nil
}

// VerifyCallback verifies and decrypts the WeChat Pay V3 callback notification.
func (a *Adapter) VerifyCallback(ctx context.Context, data *channel.CallbackData) (*channel.CallbackResult, error) {
	headers := data.Headers
	timestamp := headers["Wechatpay-Timestamp"]
	nonce := headers["Wechatpay-Nonce"]
	signature := headers["Wechatpay-Signature"]
	serial := headers["Wechatpay-Serial"]

	if timestamp == "" || nonce == "" || signature == "" || serial == "" {
		return nil, errors.New(errors.InvalidSignature, "missing wechat pay callback headers")
	}

	if err := a.verifySignature(ctx, serial, timestamp, nonce, data.RawBody, signature); err != nil {
		return nil, errors.Wrap(errors.InvalidSignature, "wechat signature verification failed", err)
	}

	var notification struct {
		ID           string `json:"id"`
		EventType    string `json:"event_type"`
		ResourceType string `json:"resource_type"`
		Resource     struct {
			Algorithm      string `json:"algorithm"`
			Ciphertext     string `json:"ciphertext"`
			AssociatedData string `json:"associated_data"`
			Nonce          string `json:"nonce"`
			OriginalType   string `json:"original_type"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(data.RawBody, &notification); err != nil {
		return nil, errors.New(errors.ValidationError, "failed to parse wechat callback json")
	}

	if notification.EventType != "TRANSACTION.SUCCESS" {
		logger.Info(ctx, "ignoring callback event", "event_type", notification.EventType, "id", notification.ID)
		return nil, fmt.Errorf("wechat: unhandled event type %s", notification.EventType)
	}

	plaintext, err := a.decryptResource(
		notification.Resource.Ciphertext,
		notification.Resource.Nonce,
		notification.Resource.AssociatedData,
	)
	if err != nil {
		return nil, errors.Wrap(errors.InvalidSignature, "wechat callback decryption failed", err)
	}

	var transaction struct {
		OutTradeNo     string `json:"out_trade_no"`
		TransactionID  string `json:"transaction_id"`
		TradeType      string `json:"trade_type"`
		TradeState     string `json:"trade_state"`
		TradeStateDesc string `json:"trade_state_desc"`
		BankType       string `json:"bank_type"`
		SuccessTime    string `json:"success_time"`
		Attach         string `json:"attach"`
		Mchid          string `json:"mchid"`
		Appid          string `json:"appid"`
		SpAppid        string `json:"sp_appid"`
		SpMchid        string `json:"sp_mchid"`
		SubAppid       string `json:"sub_appid"`
		SubMchid       string `json:"sub_mchid"`
		Payer          struct {
			Openid string `json:"openid"`
		} `json:"payer"`
		Amount struct {
			Total         int64  `json:"total"`
			PayerTotal    int64  `json:"payer_total"`
			Currency      string `json:"currency"`
			PayerCurrency string `json:"payer_currency"`
		} `json:"amount"`
		PromotionDetail json.RawMessage `json:"promotion_detail"`
	}
	if err := json.Unmarshal(plaintext, &transaction); err != nil {
		return nil, errors.New(errors.ValidationError, "failed to parse wechat transaction from callback")
	}

	logger.Info(ctx, "callback verified", "out_trade_no", transaction.OutTradeNo, "transaction_id", transaction.TransactionID, "state", transaction.TradeState, "sub_mchid", transaction.SubMchid)

	cb := &model.WechatPayCallback{
		NotificationID:      notification.ID,
		EventType:           notification.EventType,
		TransactionID:       transaction.TransactionID,
		OutTradeNo:          transaction.OutTradeNo,
		TradeType:           transaction.TradeType,
		TradeState:          transaction.TradeState,
		TradeStateDesc:      transaction.TradeStateDesc,
		BankType:            transaction.BankType,
		SuccessTime:         transaction.SuccessTime,
		Attach:              transaction.Attach,
		Mchid:               transaction.Mchid,
		Appid:               transaction.Appid,
		SpAppid:             transaction.SpAppid,
		SpMchid:             transaction.SpMchid,
		SubAppid:            transaction.SubAppid,
		SubMchid:            transaction.SubMchid,
		PayerOpenid:         transaction.Payer.Openid,
		AmountTotal:         transaction.Amount.Total,
		AmountPayerTotal:    transaction.Amount.PayerTotal,
		AmountCurrency:      transaction.Amount.Currency,
		AmountPayerCurrency: transaction.Amount.PayerCurrency,
		PromotionDetail:     []byte(transaction.PromotionDetail),
		RawBody:             string(data.RawBody),
	}

	return &channel.CallbackResult{
		ChannelTxID:    transaction.TransactionID,
		PaymentID:      transaction.OutTradeNo,
		Status:         mapWechatTradeState(transaction.TradeState),
		Amount:         transaction.Amount.Total,
		Currency:       transaction.Amount.Currency,
		WechatPayCallback: cb,
	}, nil
}

func (a *Adapter) verifySignature(ctx context.Context, serial, timestamp, nonce string, body []byte, signatureB64 string) error {
	certVisitor := downloader.MgrInstance().GetCertificateVisitor(a.mchID)
	verifier := verifiers.NewSHA256WithRSAVerifier(certVisitor)

	message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, string(body))
	err := verifier.Verify(ctx, serial, message, signatureB64)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	return nil
}

func (a *Adapter) decryptResource(ciphertextB64, nonceB64, aad string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	block, err := aes.NewCipher([]byte(a.apiV3Key))
	if err != nil {
		return nil, fmt.Errorf("aes cipher init failed: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init failed: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("aead decryption failed: %w", err)
	}
	return plaintext, nil
}

func mapWechatTradeState(state string) string {
	switch state {
	case "SUCCESS":
		return model.PaymentStatusPaid
	case "NOTPAY", "USERPAYING", "ACCEPT":
		return model.PaymentStatusPending
	case "CLOSED", "PAYERROR", "REVOKED":
		return model.PaymentStatusFailed
	case "REFUND":
		return model.PaymentStatusRefunded
	default:
		return model.PaymentStatusPending
	}
}

func getCurrency(currency string) string {
	if currency == "" {
		return "CNY"
	}
	return currency
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

func notifyURL(override, defaultURL string) string {
	if override != "" {
		return override
	}
	return defaultURL
}

