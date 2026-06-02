package unionpay

import (
	"fmt"
	"net/url"
	"strconv"
)

// CallbackResult 是回调通知解析和验签后的标准化结果。
// 对标 wechatpay-go core/notify 中的解密后内容。
type CallbackResult struct {
	QueryID            string    // 银联查询流水号（幂等键）
	OrderID            string    // 商户订单号
	TxnTime            string    // 订单发送时间 (YYYYMMDDHHmmss)
	TxnAmt             int64     // 交易金额（分）
	RespCode           string    // 响应码 ("00" = 成功)
	RespMsg            string    // 响应信息
	SettleAmt          int64     // 清算金额（分）
	SettleCurrencyCode string    // 清算币种
	SettleDate         string    // 清算日期 (MMDD)
	TraceNo            string    // 系统跟踪号
	TraceTime          string    // 交易传输时间
	ExchangeRate       string    // 汇率
	AccNo              string    // 账号掩码
	PayCardType        string    // 支付卡类型
	Signature          string    // 银联应答签名
	SignMethod         string    // 签名方法
	RawValues          url.Values // 所有原始参数
}

// NotifyHandler 是回调通知处理器，负责验签和解析。
// 对标 wechatpay-go core/notify.Handler。
type NotifyHandler struct {
	client *Client
}

// NewNotifyHandler 创建通知处理器。
// 需要 Client 带有公钥（PublicKey）才能验签。
func NewNotifyHandler(client *Client) *NotifyHandler {
	return &NotifyHandler{client: client}
}

// Parse 解析回调通知 body（form-encoded 格式），验证签名后返回标准化结果。
// 对标 wechatpay-go Handler.ParseNotifyRequest()。
//
// body 应为回调请求的原始 HTTP body 字节。
func (h *NotifyHandler) Parse(body []byte) (*CallbackResult, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("unionpay-go: failed to parse callback body: %w", err)
	}

	if h.client.publicKey == nil {
		return nil, fmt.Errorf("unionpay-go: public key not configured, cannot verify callback")
	}

	signature := values.Get("signature")
	if signature == "" {
		return nil, fmt.Errorf("unionpay-go: missing signature in callback")
	}

	if err := h.client.verifySign(values); err != nil {
		return nil, fmt.Errorf("unionpay-go: signature verification failed: %w", err)
	}

	orderID := values.Get("orderId")
	queryID := values.Get("queryId")

	if orderID == "" {
		return nil, fmt.Errorf("unionpay-go: missing orderId in callback")
	}
	if queryID == "" {
		return nil, fmt.Errorf("unionpay-go: missing queryId in callback")
	}

	txnAmt, _ := strconv.ParseInt(values.Get("txnAmt"), 10, 64)
	settleAmt, _ := strconv.ParseInt(values.Get("settleAmt"), 10, 64)

	return &CallbackResult{
		QueryID:            queryID,
		OrderID:            orderID,
		TxnTime:            values.Get("txnTime"),
		TxnAmt:             txnAmt,
		RespCode:           values.Get("respCode"),
		RespMsg:            values.Get("respMsg"),
		SettleAmt:          settleAmt,
		SettleCurrencyCode: values.Get("settleCurrencyCode"),
		SettleDate:         values.Get("settleDate"),
		TraceNo:            values.Get("traceNo"),
		TraceTime:          values.Get("traceTime"),
		ExchangeRate:       values.Get("exchangeRate"),
		AccNo:              values.Get("accNo"),
		PayCardType:        values.Get("payCardType"),
		Signature:          signature,
		SignMethod:         values.Get("signMethod"),
		RawValues:          values,
	}, nil
}
