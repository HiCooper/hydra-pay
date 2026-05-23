package wechat

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/app"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	partnerapp "github.com/wechatpay-apiv3/wechatpay-go/services/partnerpayments/app"
	partnerjsapi "github.com/wechatpay-apiv3/wechatpay-go/services/partnerpayments/jsapi"
	partnernative "github.com/wechatpay-apiv3/wechatpay-go/services/partnerpayments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/pkg/errors"
)

// Adapter implements the channel.Adapter interface for WeChat Pay V3.
// Supports both direct merchant and service provider (partner) modes.
type Adapter struct {
	client          *core.Client
	mchID           string
	apiV3Key        string
	notifyURL       string
	nativeSvc       *native.NativeApiService
	jsapiSvc        *jsapi.JsapiApiService
	appSvc          *app.AppApiService
	partnerNativeSvc *partnernative.NativeApiService
	partnerJsapiSvc  *partnerjsapi.JsapiApiService
	partnerAppSvc    *partnerapp.AppApiService
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

	log.Printf("[wechat] adapter initialized (mch_id=%s, serial=%s)", cfg.MchID, cfg.SerialNo)
	return &Adapter{
		client:           client,
		mchID:            cfg.MchID,
		apiV3Key:         cfg.APIv3Key,
		notifyURL:        cfg.NotifyURL,
		nativeSvc:        &native.NativeApiService{Client: client},
		jsapiSvc:         &jsapi.JsapiApiService{Client: client},
		appSvc:           &app.AppApiService{Client: client},
		partnerNativeSvc: &partnernative.NativeApiService{Client: client},
		partnerJsapiSvc:  &partnerjsapi.JsapiApiService{Client: client},
		partnerAppSvc:    &partnerapp.AppApiService{Client: client},
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

	// Service provider mode: use partner APIs
	if req.SubMerchantID != "" {
		return a.createPartnerPayment(ctx, req)
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

func (a *Adapter) createPartnerPayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	switch req.TradeType {
	case "native":
		return a.createPartnerNativePayment(ctx, req)
	case "jsapi", "miniapp":
		return a.createPartnerJSAPIPayment(ctx, req)
	case "app":
		return a.createPartnerAppPayment(ctx, req)
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
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat native prepay failed", err)
	}

	log.Printf("[wechat] native prepay success: out_trade_no=%s", req.PaymentID)
	return &channel.CreatePaymentResponse{
		ChannelTxID: req.PaymentID,
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
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat jsapi prepay failed", err)
	}

	log.Printf("[wechat] jsapi prepay success: out_trade_no=%s, prepay_id=%s", req.PaymentID, *resp.PrepayId)
	return &channel.CreatePaymentResponse{
		ChannelTxID: req.PaymentID,
		PaymentURL:  *resp.PrepayId,
		RawResponse: structToMap(resp),
	}, nil
}

func (a *Adapter) createAppPayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	appid := req.ChannelAppID
	if appid == "" {
		return nil, errors.New(errors.ValidationError, "channel_app_id is required for app payment")
	}

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
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat app prepay failed", err)
	}

	log.Printf("[wechat] app prepay success: out_trade_no=%s, prepay_id=%s", req.PaymentID, *resp.PrepayId)
	return &channel.CreatePaymentResponse{
		ChannelTxID: req.PaymentID,
		PaymentURL:  *resp.PrepayId,
		RawResponse: structToMap(resp),
	}, nil
}

// --- Service provider (partner) payments ---

func (a *Adapter) createPartnerNativePayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	if req.ChannelAppID == "" {
		return nil, errors.New(errors.ValidationError, "channel_app_id (sp_appid) is required for partner native payment")
	}

	resp, _, err := a.partnerNativeSvc.Prepay(ctx,
		partnernative.PrepayRequest{
			SpAppid:     core.String(req.ChannelAppID),
			SpMchid:     core.String(a.mchID),
			SubMchid:    core.String(req.SubMerchantID),
			SubAppid:    toCoreString(req.SubChannelAppID),
			Description: core.String(truncate(req.Description, 127)),
			OutTradeNo:  core.String(req.PaymentID),
			NotifyUrl:   core.String(notifyURL(req.NotifyURL, a.notifyURL)),
			Amount: &partnernative.Amount{
				Currency: core.String(getCurrency(req.Currency)),
				Total:    core.Int64(req.Amount),
			},
		},
	)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat partner native prepay failed", err)
	}

	log.Printf("[wechat] partner native prepay success: out_trade_no=%s, sub_mchid=%s", req.PaymentID, req.SubMerchantID)
	return &channel.CreatePaymentResponse{
		ChannelTxID: req.PaymentID,
		QRCodeURL:   *resp.CodeUrl,
		RawResponse: map[string]interface{}{"code_url": *resp.CodeUrl},
	}, nil
}

func (a *Adapter) createPartnerJSAPIPayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	if req.ChannelAppID == "" {
		return nil, errors.New(errors.ValidationError, "channel_app_id (sp_appid) is required for partner jsapi payment")
	}
	if req.OpenID == "" {
		return nil, errors.New(errors.ValidationError, "open_id is required for jsapi payment")
	}

	resp, _, err := a.partnerJsapiSvc.Prepay(ctx,
		partnerjsapi.PrepayRequest{
			SpAppid:     core.String(req.ChannelAppID),
			SpMchid:     core.String(a.mchID),
			SubMchid:    core.String(req.SubMerchantID),
			SubAppid:    toCoreString(req.SubChannelAppID),
			Description: core.String(truncate(req.Description, 127)),
			OutTradeNo:  core.String(req.PaymentID),
			NotifyUrl:   core.String(notifyURL(req.NotifyURL, a.notifyURL)),
			Amount: &partnerjsapi.Amount{
				Currency: core.String(getCurrency(req.Currency)),
				Total:    core.Int64(req.Amount),
			},
			Payer: &partnerjsapi.Payer{SubOpenid: core.String(req.OpenID)},
		},
	)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat partner jsapi prepay failed", err)
	}

	log.Printf("[wechat] partner jsapi prepay success: out_trade_no=%s, sub_mchid=%s", req.PaymentID, req.SubMerchantID)
	return &channel.CreatePaymentResponse{
		ChannelTxID: req.PaymentID,
		PaymentURL:  *resp.PrepayId,
		RawResponse: structToMap(resp),
	}, nil
}

func (a *Adapter) createPartnerAppPayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	if req.ChannelAppID == "" {
		return nil, errors.New(errors.ValidationError, "channel_app_id (sp_appid) is required for partner app payment")
	}

	resp, _, err := a.partnerAppSvc.Prepay(ctx,
		partnerapp.PrepayRequest{
			SpAppid:     core.String(req.ChannelAppID),
			SpMchid:     core.String(a.mchID),
			SubMchid:    core.String(req.SubMerchantID),
			SubAppid:    toCoreString(req.SubChannelAppID),
			Description: core.String(truncate(req.Description, 127)),
			OutTradeNo:  core.String(req.PaymentID),
			NotifyUrl:   core.String(notifyURL(req.NotifyURL, a.notifyURL)),
			Amount:      &partnerapp.Amount{Currency: core.String(getCurrency(req.Currency)), Total: core.Int64(req.Amount)},
		},
	)
	if err != nil {
		return nil, errors.Wrap(errors.ChannelError, "wechat partner app prepay failed", err)
	}

	log.Printf("[wechat] partner app prepay success: out_trade_no=%s, sub_mchid=%s", req.PaymentID, req.SubMerchantID)
	return &channel.CreatePaymentResponse{
		ChannelTxID: req.PaymentID,
		PaymentURL:  *resp.PrepayId,
		RawResponse: structToMap(resp),
	}, nil
}

// GetPaymentStatus queries the order status by out_trade_no.
func (a *Adapter) GetPaymentStatus(ctx context.Context, channelTxID string) (string, error) {
	resp, _, err := a.nativeSvc.QueryOrderByOutTradeNo(ctx,
		native.QueryOrderByOutTradeNoRequest{
			OutTradeNo: core.String(channelTxID),
			Mchid:      core.String(a.mchID),
		},
	)
	if err != nil {
		return "", errors.Wrap(errors.ChannelError, "wechat query order failed", err)
	}

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
		log.Printf("[wechat] ignoring callback event: %s (id=%s)", notification.EventType, notification.ID)
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
		TradeState     string `json:"trade_state"`
		TradeStateDesc string `json:"trade_state_desc"`
		SubMchid       string `json:"sub_mchid"`
		SpMchid        string `json:"sp_mchid"`
		Amount         struct {
			Total    int64  `json:"total"`
			Currency string `json:"currency"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(plaintext, &transaction); err != nil {
		return nil, errors.New(errors.ValidationError, "failed to parse wechat transaction from callback")
	}

	log.Printf("[wechat] callback verified: out_trade_no=%s, transaction_id=%s, state=%s, sub_mchid=%s",
		transaction.OutTradeNo, transaction.TransactionID, transaction.TradeState, transaction.SubMchid)

	return &channel.CallbackResult{
		ChannelTxID: transaction.TransactionID,
		PaymentID:   transaction.OutTradeNo,
		Status:      mapWechatTradeState(transaction.TradeState),
		Amount:      transaction.Amount.Total,
		Currency:    transaction.Amount.Currency,
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

func toCoreString(s string) *string {
	if s == "" {
		return nil
	}
	return core.String(s)
}