package unionpay

import (
	"context"
	"fmt"
	"time"

	sentinel "github.com/alibaba/sentinel-golang/api"
	unionpay "github.com/hydra/unionpay-go"
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

// Adapter 实现 channel.Adapter 接口，对接银联云闪付。
// 使用 github.com/hydra/unionpay-go SDK 处理银联 API 通信，
// 本层只负责 Sentinel 熔断、OpenTelemetry 追踪、Prometheus 指标
// 和 hydra-pay 模型映射等业务集成逻辑。
type Adapter struct {
	client    *unionpay.Client
	paySvc    *unionpay.PayService
	querySvc  *unionpay.QueryService
	refundSvc *unionpay.RefundService
	notifyH   *unionpay.NotifyHandler
	notifyURL string
	returnURL string
	isSandbox bool
}

// NewAdapter 创建云闪付渠道适配器。
// 对标 channel/alipay 和 channel/wechat 的 NewAdapter 模式。
func NewAdapter(cfg *config.UnionpayConfig) (*Adapter, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("unionpay: UNIONPAY_APP_ID is required")
	}
	if cfg.Secret == "" {
		return nil, fmt.Errorf("unionpay: UNIONPAY_SECRET is required")
	}
	if cfg.MchID == "" {
		return nil, fmt.Errorf("unionpay: UNIONPAY_MCH_ID is required")
	}

	privKey, err := unionpay.LoadPrivateKeyPEM(cfg.PrivateKey, cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("unionpay: failed to load private key: %w", err)
	}

	clientOpts := []unionpay.Option{
		unionpay.WithMchID(cfg.MchID),
		unionpay.WithPrivateKey(privKey),
		unionpay.WithSandbox(cfg.IsSandbox),
	}

	if cfg.UnionpayPublicKey != "" || cfg.UnionpayPublicKeyPath != "" {
		pk, err := unionpay.LoadPublicKeyPEM(cfg.UnionpayPublicKey, cfg.UnionpayPublicKeyPath)
		if err != nil {
			logger.Warn(context.Background(), "unionpay: failed to load public key, callback verification disabled",
				"error", err)
		} else {
			clientOpts = append(clientOpts, unionpay.WithPublicKey(pk))
		}
	} else {
		logger.Warn(context.Background(), "unionpay: no public key configured — callback verification will fail")
	}

	client, err := unionpay.NewClient(context.Background(), clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("unionpay: failed to create client: %w", err)
	}

	logger.Info(context.Background(), "unionpay adapter initialized",
		"app_id", cfg.AppID, "mch_id", cfg.MchID, "sandbox", cfg.IsSandbox)

	return &Adapter{
		client:    client,
		paySvc:    &unionpay.PayService{Service: unionpay.Service{Client: client}},
		querySvc:  &unionpay.QueryService{Service: unionpay.Service{Client: client}},
		refundSvc: &unionpay.RefundService{Service: unionpay.Service{Client: client}},
		notifyH:   unionpay.NewNotifyHandler(client),
		notifyURL: cfg.NotifyURL,
		returnURL: cfg.ReturnURL,
		isSandbox: cfg.IsSandbox,
	}, nil
}

func (a *Adapter) Name() string { return model.ChannelUnionpay }

// CreatePayment 创建支付，按 TradeType 分发。
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
	case "jsapi", "h5":
		return a.createH5(ctx, req, backURL)
	default:
		return nil, errors.New(errors.ValidationError, "unsupported unionpay trade type: "+req.TradeType)
	}
}

func (a *Adapter) createNative(ctx context.Context, req *channel.CreatePaymentRequest, backURL string) (*channel.CreatePaymentResponse, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "unionpay.native",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "unionpay"), attribute.String("operation", "native")),
	)
	defer span.End()

	e, b := sentinel.Entry("unionpay")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "unionpay circuit breaker open")
	}

	start := time.Now()
	resp, err := a.paySvc.QRCodePay(ctx, &unionpay.QRCodePayReq{
		OrderID:  req.PaymentID,
		TxnAmt:   req.Amount,
		BackURL:  backURL,
		SubMerID: req.SubMerchantID,
	})
	metrics.ChannelAPIRequestDuration.WithLabelValues("unionpay", "native").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("unionpay", "native", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "unionpay native payment failed", err)
	}
	e.Exit()

	logger.Info(ctx, "unionpay native payment created", "order_id", req.PaymentID, "query_id", resp.QueryID)

	return &channel.CreatePaymentResponse{
		ChannelTxID: resp.QueryID,
		QRCodeURL:   resp.QRCode,
		RawResponse: map[string]interface{}{
			"query_id": resp.QueryID, "qr_code": resp.QRCode, "resp_code": resp.RespCode},
	}, nil
}

func (a *Adapter) createApp(ctx context.Context, req *channel.CreatePaymentRequest, backURL string) (*channel.CreatePaymentResponse, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "unionpay.app",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "unionpay"), attribute.String("operation", "app")),
	)
	defer span.End()

	e, b := sentinel.Entry("unionpay")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "unionpay circuit breaker open")
	}

	start := time.Now()
	resp, err := a.paySvc.AppPay(ctx, &unionpay.AppPayReq{
		OrderID:  req.PaymentID,
		TxnAmt:   req.Amount,
		BackURL:  backURL,
		SubMerID: req.SubMerchantID,
	})
	metrics.ChannelAPIRequestDuration.WithLabelValues("unionpay", "app").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("unionpay", "app", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "unionpay app payment failed", err)
	}
	e.Exit()

	logger.Info(ctx, "unionpay app payment created", "order_id", req.PaymentID, "tn", resp.TN)

	return &channel.CreatePaymentResponse{
		ChannelTxID: resp.QueryID,
		PaymentURL:  resp.TN,
		RawResponse: map[string]interface{}{"query_id": resp.QueryID, "tn": resp.TN},
	}, nil
}

func (a *Adapter) createH5(ctx context.Context, req *channel.CreatePaymentRequest, backURL string) (*channel.CreatePaymentResponse, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "unionpay.h5",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "unionpay"), attribute.String("operation", "h5")),
	)
	defer span.End()

	e, b := sentinel.Entry("unionpay")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "unionpay circuit breaker open")
	}

	start := time.Now()
	htmlForm, err := a.paySvc.H5Pay(ctx, &unionpay.H5PayReq{
		OrderID:  req.PaymentID,
		TxnAmt:   req.Amount,
		BackURL:  backURL,
		FrontURL: a.returnURL,
		SubMerID: req.SubMerchantID,
	})
	metrics.ChannelAPIRequestDuration.WithLabelValues("unionpay", "h5").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("unionpay", "h5", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "unionpay h5 payment failed", err)
	}
	e.Exit()

	logger.Info(ctx, "unionpay h5 payment created", "order_id", req.PaymentID)

	return &channel.CreatePaymentResponse{
		PaymentURL:  htmlForm,
		RawResponse: map[string]interface{}{"html_form": htmlForm},
	}, nil
}

// VerifyCallback 验证回调签名并返回标准化结果。
func (a *Adapter) VerifyCallback(ctx context.Context, data *channel.CallbackData) (*channel.CallbackResult, error) {
	result, err := a.notifyH.Parse(data.RawBody)
	if err != nil {
		return nil, errors.Wrap(errors.InvalidSignature, "unionpay callback verification failed", err)
	}

	logger.Info(ctx, "unionpay callback verified",
		"order_id", result.OrderID, "query_id", result.QueryID, "resp_code", result.RespCode)

	status := mapRespCode(result.RespCode)

	// Double-check by querying UnionPay API (anti-replay)
	if status == model.PaymentStatusPaid {
		qResp, qErr := a.querySvc.QueryOrder(ctx, &unionpay.QueryReq{OrderID: result.OrderID})
		if qErr != nil {
			logger.Error(ctx, "callback double-check query failed", "error", qErr)
		} else if qResp.OrigRespCode != "" {
			status = mapRespCode(qResp.OrigRespCode)
		}
	}

	cb := &model.UnionpayCallback{
		QueryID:            result.QueryID,
		OrderID:            result.OrderID,
		TxnTime:            result.TxnTime,
		TxnAmt:             result.TxnAmt,
		RespCode:           result.RespCode,
		RespMsg:            result.RespMsg,
		SettleAmt:          result.SettleAmt,
		SettleCurrencyCode: result.SettleCurrencyCode,
		SettleDate:         result.SettleDate,
		TraceNo:            result.TraceNo,
		TraceTime:          result.TraceTime,
		ExchangeRate:       result.ExchangeRate,
		AccNo:              result.AccNo,
		PayCardType:        result.PayCardType,
		Signature:          result.Signature,
		SignMethod:         result.SignMethod,
		RawBody:            string(data.RawBody),
	}

	return &channel.CallbackResult{
		ChannelTxID:     result.QueryID,
		PaymentID:       result.OrderID,
		Status:          status,
		Amount:          result.TxnAmt,
		Currency:        "CNY",
		UnionpayCallback: cb,
	}, nil
}

// GetPaymentStatus 查询订单状态。
func (a *Adapter) GetPaymentStatus(ctx context.Context, channelTxID string) (string, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "unionpay.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "unionpay"), attribute.String("operation", "query")),
	)
	defer span.End()

	e, b := sentinel.Entry("unionpay")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return "", errors.New(errors.ChannelError, "unionpay circuit breaker open")
	}

	start := time.Now()
	resp, err := a.querySvc.QueryOrder(ctx, &unionpay.QueryReq{OrderID: channelTxID})
	metrics.ChannelAPIRequestDuration.WithLabelValues("unionpay", "query").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("unionpay", "query", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return "", errors.Wrap(errors.ChannelError, "unionpay query order failed", err)
	}
	e.Exit()

	origCode := resp.OrigRespCode
	if origCode == "" {
		origCode = "00"
	}
	return mapRespCode(origCode), nil
}

// Refund 发起退款。
func (a *Adapter) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	ctx, span := otel.Tracer("hydra-pay").Start(ctx, "unionpay.refund",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("channel", "unionpay"), attribute.String("operation", "refund")),
	)
	defer span.End()

	e, b := sentinel.Entry("unionpay")
	if b != nil {
		span.SetStatus(codes.Error, "circuit breaker open")
		return nil, errors.New(errors.ChannelError, "unionpay circuit breaker open")
	}

	start := time.Now()
	resp, err := a.refundSvc.Refund(ctx, &unionpay.RefundReq{
		OrderID:   req.TradeNo,
		OrigQryID: req.ChannelTxID,
		TxnAmt:    req.RefundAmount,
		BackURL:   a.notifyURL,
		SubMerID:  req.SubMerchantID,
	})
	metrics.ChannelAPIRequestDuration.WithLabelValues("unionpay", "refund").Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.ChannelAPIRequestTotal.WithLabelValues("unionpay", "refund", "error").Inc()
		sentinel.TraceError(e, err)
		e.Exit()
		span.SetStatus(codes.Error, err.Error())
		return nil, errors.Wrap(errors.ChannelError, "unionpay refund failed", err)
	}
	e.Exit()

	logger.Info(ctx, "unionpay refund success", "order_id", req.TradeNo, "refund_id", resp.QueryID)

	return &channel.RefundResponse{
		ChannelRefundID: resp.QueryID,
		RefundFee:       resp.TxnAmt,
		RawResponse:     map[string]interface{}{"query_id": resp.QueryID, "txn_amt": resp.TxnAmt},
	}, nil
}

// ---- Helpers ----

func mapRespCode(code string) string {
	switch code {
	case "00":
		return model.PaymentStatusPaid
	case "03", "04", "05":
		return model.PaymentStatusPending
	default:
		return model.PaymentStatusFailed
	}
}


