package ecny

import (
	"context"
	"fmt"
	"time"

	sentinel "github.com/alibaba/sentinel-golang/api"
	ecny "github.com/hydra/ecny-go"
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

// Adapter 实现 channel.Adapter 接口，对接数字人民币（e-CNY）受理服务机构。
// 使用 github.com/hydra/ecny-go SDK 处理数字人民币 API 通信，
// 本层只负责 Sentinel 熔断、OpenTelemetry 追踪、Prometheus 指标
// 和 hydra-pay 模型映射等业务集成逻辑。
//
// 对标 channel/unionpay 的 Adapter 模式。
type Adapter struct {
	client    *ecny.Client
	paySvc    *ecny.PayService
	querySvc  *ecny.QueryService
	refundSvc *ecny.RefundService
	notifyH   *ecny.NotifyHandler
	notifyURL string
	returnURL string
	isSandbox bool
	appID     string
}

// NewAdapter 创建数字人民币渠道适配器。
// 对标 channel/alipay 和 channel/unionpay 的 NewAdapter 模式。
func NewAdapter(cfg *config.EcnyConfig) (*Adapter, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("ecny: ECNY_APP_ID is required")
	}
	if cfg.MchID == "" {
		return nil, fmt.Errorf("ecny: ECNY_MCH_ID is required")
	}

	privKey, err := ecny.LoadSM2PrivateKeyPEM(cfg.PrivateKey, cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("ecny: failed to load SM2 private key: %w", err)
	}

	clientOpts := []ecny.Option{
		ecny.WithAppID(cfg.AppID),
		ecny.WithMchID(cfg.MchID),
		ecny.WithPrivateKey(privKey),
		ecny.WithSandbox(cfg.IsSandbox),
	}

	// Override agency API base URL if configured
	if cfg.AgencyAPIBaseURL != "" {
		clientOpts = append(clientOpts, ecny.WithBaseURL(cfg.AgencyAPIBaseURL))
	}

	// Load agency SM2 public key for callback verification
	if cfg.AgencyPublicKey != "" || cfg.AgencyPublicKeyPath != "" {
		pubKey, err := ecny.LoadSM2PublicKeyPEM(cfg.AgencyPublicKey, cfg.AgencyPublicKeyPath)
		if err != nil {
			logger.Warn(context.Background(), "ecny: failed to load agency public key, callback verification disabled",
				"error", err)
		} else {
			clientOpts = append(clientOpts, ecny.WithPublicKey(pubKey))
		}
	} else {
		logger.Warn(context.Background(), "ecny: no agency public key configured — callback verification will fail")
	}

	client, err := ecny.NewClient(context.Background(), clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("ecny: failed to create client: %w", err)
	}

	logger.Info(context.Background(), "ecny adapter initialized",
		"app_id", cfg.AppID, "mch_id", cfg.MchID, "sandbox", cfg.IsSandbox)

	return &Adapter{
		client:    client,
		paySvc:    &ecny.PayService{Service: ecny.Service{Client: client}},
		querySvc:  &ecny.QueryService{Service: ecny.Service{Client: client}},
		refundSvc: &ecny.RefundService{Service: ecny.Service{Client: client}},
		notifyH:   ecny.NewNotifyHandler(client),
		notifyURL: cfg.NotifyURL,
		returnURL: cfg.ReturnURL,
		isSandbox: cfg.IsSandbox,
		appID:     cfg.AppID,
	}, nil
}

func (a *Adapter) Name() string { return model.ChannelEcny }

// CreatePayment 创建支付，按 TradeType 分发。
// 支持两种交易类型:
//   - "native": 扫码支付（主扫），返回二维码 URL 供用户使用数字人民币 APP 扫码
//   - "app": 拉起支付，返回加密订单信息供客户端唤起数字人民币收银台（对标瑞幸模式）
func (a *Adapter) CreatePayment(ctx context.Context, req *channel.CreatePaymentRequest) (*channel.CreatePaymentResponse, error) {
	if req.Amount <= 0 {
		return nil, errors.New(errors.ValidationError, "amount must be positive")
	}
	if req.TradeType == "" {
		req.TradeType = "native"
	}

	backURL := a.notifyURL
	if req.NotifyURL != "" {
		backURL = req.NotifyURL
	}

	switch req.TradeType {
	case "native":
		return a.createNative(ctx, req, backURL)
	case "app":
		return a.createApp(ctx, req, backURL)
	default:
		return nil, errors.New(errors.ValidationError, "unsupported ecny trade type: "+req.TradeType)
	}
}

func (a *Adapter) createNative(ctx context.Context, req *channel.CreatePaymentRequest, backURL string) (*channel.CreatePaymentResponse, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "ecny.native",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "ecny"), attribute.String("operation", "native")),
	)
	defer span.End()

	e, b := sentinel.Entry("ecny")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "ecny circuit breaker open")
	}

	start := time.Now()
	resp, err := a.paySvc.QRCodePay(ctx, &ecny.QRCodePayReq{
		OrderID:  req.PaymentID,
		TxnAmt:   req.Amount,
		BackURL:  backURL,
		Subject:  req.Description,
	})
	metrics.ChannelAPIRequestDuration.WithLabelValues("ecny", "native").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("ecny", "native", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "ecny native payment failed", err)
	}
	e.Exit()

	logger.Info(ctx, "ecny native payment created", "order_id", req.PaymentID, "channel_tx_id", resp.ChannelTxID)

	return &channel.CreatePaymentResponse{
		ChannelTxID: resp.ChannelTxID,
		QRCodeURL:   resp.QRCodeURL,
		RawResponse: map[string]interface{}{
			"channel_tx_id": resp.ChannelTxID,
			"qr_code":       resp.QRCodeURL,
			"resp_code":     resp.RespCode,
		},
	}, nil
}

func (a *Adapter) createApp(ctx context.Context, req *channel.CreatePaymentRequest, backURL string) (*channel.CreatePaymentResponse, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "ecny.app",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "ecny"), attribute.String("operation", "app")),
	)
	defer span.End()

	e, b := sentinel.Entry("ecny")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "ecny circuit breaker open")
	}

	start := time.Now()
	resp, err := a.paySvc.AppPay(ctx, &ecny.AppPayReq{
		OrderID:  req.PaymentID,
		TxnAmt:   req.Amount,
		BackURL:  backURL,
		Subject:  req.Description,
	})
	metrics.ChannelAPIRequestDuration.WithLabelValues("ecny", "app").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("ecny", "app", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "ecny app payment failed", err)
	}
	e.Exit()

	logger.Info(ctx, "ecny app payment created", "order_id", req.PaymentID, "channel_tx_id", resp.ChannelTxID)

	return &channel.CreatePaymentResponse{
		ChannelTxID: resp.ChannelTxID,
		RawResponse: map[string]interface{}{
			"channel_tx_id":  resp.ChannelTxID,
			"encrypted_key":  resp.EncryptedKey,
			"encrypted_info": resp.EncryptedInfo,
			"encryption_sn":  resp.EncryptionSN,
			"extra_info":     resp.ExtraInfo,
		},
	}, nil
}

// VerifyCallback 验证回调 SM2 签名并返回标准化结果。
func (a *Adapter) VerifyCallback(ctx context.Context, data *channel.CallbackData) (*channel.CallbackResult, error) {
	result, err := a.notifyH.Parse(data.RawBody)
	if err != nil {
		return nil, errors.Wrap(errors.InvalidSignature, "ecny callback verification failed", err)
	}

	logger.Info(ctx, "ecny callback verified",
		"order_id", result.OrderID, "channel_tx_id", result.ChannelTxID, "status", result.Status)

	status := mapEcnyStatus(result.Status)

	// Double-check by querying the agency API (anti-replay)
	if status == model.PaymentStatusPaid {
		qResp, qErr := a.querySvc.QueryOrder(ctx, &ecny.QueryReq{
			OrderID:     result.OrderID,
			ChannelTxID: result.ChannelTxID,
		})
		if qErr != nil {
			logger.Error(ctx, "ecny callback double-check query failed", "error", qErr)
		} else if qResp.Status != "" {
			status = mapEcnyStatus(qResp.Status)
		}
	}

	cb := &model.EcnyCallback{
		ChannelTxID: result.ChannelTxID,
		OrderID:     result.OrderID,
		TxnAmt:      result.TxnAmt,
		TxnTime:     result.TxnTime,
		RespCode:    result.RespCode,
		RespMsg:     result.RespMsg,
		Status:      result.Status,
		Signature:   result.Signature,
		SignMethod:  result.SignMethod,
		RawBody:     result.RawBody,
	}

	return &channel.CallbackResult{
		ChannelTxID:  result.ChannelTxID,
		PaymentID:    result.OrderID,
		Status:       status,
		Amount:       result.TxnAmt,
		Currency:     "CNY",
		EcnyCallback: cb,
	}, nil
}

// GetPaymentStatus 查询订单状态。
func (a *Adapter) GetPaymentStatus(ctx context.Context, channelTxID string) (string, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "ecny.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "ecny"), attribute.String("operation", "query")),
	)
	defer span.End()

	e, b := sentinel.Entry("ecny")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return "", errors.New(errors.ChannelError, "ecny circuit breaker open")
	}

	start := time.Now()
	resp, err := a.querySvc.QueryOrder(ctx, &ecny.QueryReq{
		OrderID:     channelTxID,
		ChannelTxID: channelTxID,
	})
	metrics.ChannelAPIRequestDuration.WithLabelValues("ecny", "query").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("ecny", "query", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return "", errors.Wrap(errors.ChannelError, "ecny query order failed", err)
	}
	e.Exit()

	return mapEcnyStatus(resp.Status), nil
}

// Refund 发起退款。
func (a *Adapter) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "ecny.refund",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "ecny"), attribute.String("operation", "refund")),
	)
	defer span.End()

	e, b := sentinel.Entry("ecny")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "ecny circuit breaker open")
	}

	start := time.Now()
	resp, err := a.refundSvc.Refund(ctx, &ecny.RefundReq{
		OrderID:      req.TradeNo,
		ChannelTxID:  req.ChannelTxID,
		RefundAmount: req.RefundAmount,
		RefundReason: req.RefundReason,
		BackURL:      a.notifyURL,
	})
	metrics.ChannelAPIRequestDuration.WithLabelValues("ecny", "refund").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("ecny", "refund", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "ecny refund failed", err)
	}
	e.Exit()

	logger.Info(ctx, "ecny refund success", "order_id", req.TradeNo, "refund_id", resp.ChannelRefundID)

	return &channel.RefundResponse{
		ChannelRefundID: resp.ChannelRefundID,
		RefundFee:       resp.RefundAmount,
		RawResponse: map[string]interface{}{
			"channel_refund_id": resp.ChannelRefundID,
			"refund_amount":     resp.RefundAmount,
		},
	}, nil
}

// ---- Helpers ----

func mapEcnyStatus(status string) string {
	switch status {
	case "SUCCESS", "TRADE_SUCCESS", "00", "0000":
		return model.PaymentStatusPaid
	case "PROCESSING", "WAIT_PAY", "USERPAYING":
		return model.PaymentStatusPending
	case "CLOSED", "TRADE_CLOSED", "FAILED", "PAYERROR":
		return model.PaymentStatusFailed
	default:
		return model.PaymentStatusPending
	}
}
