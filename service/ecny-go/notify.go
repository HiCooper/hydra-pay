package ecny

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// CallbackResult 是回调通知解析和验签后的标准化结果。
// 对标 unionpay-go CallbackResult。
type CallbackResult struct {
	ChannelTxID string // 受理机构交易流水号（幂等键）
	OrderID     string // 商户订单号
	TxnAmt      int64  // 交易金额（分）
	TxnTime     string // 交易时间
	Status      string // 交易状态: SUCCESS, PROCESSING, CLOSED, FAILED
	RespCode    string // 响应码
	RespMsg     string // 响应信息
	Signature   string // 回调签名（Base64）
	SignMethod  string // 签名方法: "SM2"
	RawBody     string // 原始回调 JSON 体
}

// NotifyHandler 是回调通知处理器，负责验签和解析。
// 对标 unionpay-go NotifyHandler。
type NotifyHandler struct {
	client *Client
}

// NewNotifyHandler 创建通知处理器。
// 需要 Client 带有公钥（PublicKey）才能验签。
func NewNotifyHandler(client *Client) *NotifyHandler {
	return &NotifyHandler{client: client}
}

// Parse 解析回调通知 JSON body，验证 SM2 签名后返回标准化结果。
//
// 回调 JSON 格式（受理服务机构通用格式）：
//
//	{
//	  "channelTxId": "受理机构交易流水号",
//	  "orderId": "商户订单号",
//	  "txnAmt": 金额(分),
//	  "status": "SUCCESS",
//	  "respCode": "00",
//	  "respMsg": "成功",
//	  "sign": "Base64(SM2签名)",
//	  "signMethod": "SM2"
//	}
//
// 签名验证流程：
//  1. 从 JSON 中提取 sign 字段
//  2. 用其余字段按 key 排序构建签名字符串
//  3. 使用受理机构 SM2 公钥验证签名
func (h *NotifyHandler) Parse(body []byte) (*CallbackResult, error) {
	if h.client.publicKey == nil {
		return nil, fmt.Errorf("ecny-go: public key not configured, cannot verify callback")
	}

	// Parse JSON body into a map for signing string building
	var rawMap map[string]interface{}
	if err := json.Unmarshal(body, &rawMap); err != nil {
		return nil, fmt.Errorf("ecny-go: failed to parse callback body: %w", err)
	}

	signB64, _ := rawMap["sign"].(string)
	if signB64 == "" {
		signB64, _ = rawMap["signature"].(string)
	}
	if signB64 == "" {
		return nil, fmt.Errorf("ecny-go: missing sign/signature in callback")
	}

	// Build signing string from all fields except sign/signature/signMethod
	signParams := make(map[string]string)
	for k, v := range rawMap {
		if k == "sign" || k == "signature" || k == "signMethod" {
			continue
		}
		signParams[k] = fmt.Sprintf("%v", v)
	}

	signingStr := BuildSigningString(signParams)
	if err := h.client.verifySign(signingStr, signB64); err != nil {
		return nil, fmt.Errorf("ecny-go: callback signature verification failed: %w", err)
	}

	orderID, _ := rawMap["orderId"].(string)
	channelTxID, _ := rawMap["channelTxId"].(string)
	if channelTxID == "" {
		channelTxID, _ = rawMap["transactionId"].(string)
	}
	if orderID == "" {
		return nil, fmt.Errorf("ecny-go: missing orderId in callback")
	}
	if channelTxID == "" {
		return nil, fmt.Errorf("ecny-go: missing channelTxId/transactionId in callback")
	}

	status, _ := rawMap["status"].(string)
	if status == "" {
		status, _ = rawMap["tradeState"].(string)
	}

	var txnAmt int64
	switch v := rawMap["txnAmt"].(type) {
	case float64:
		txnAmt = int64(v)
	case string:
		txnAmt, _ = strconv.ParseInt(v, 10, 64)
	}

	txnTime, _ := rawMap["txnTime"].(string)
	respCode, _ := rawMap["respCode"].(string)
	respMsg, _ := rawMap["respMsg"].(string)
	signMethod, _ := rawMap["signMethod"].(string)
	if signMethod == "" {
		signMethod = SignMethodSM2
	}

	return &CallbackResult{
		ChannelTxID: channelTxID,
		OrderID:     orderID,
		TxnAmt:      txnAmt,
		TxnTime:     txnTime,
		Status:      status,
		RespCode:    respCode,
		RespMsg:     respMsg,
		Signature:   signB64,
		SignMethod:  signMethod,
		RawBody:     string(body),
	}, nil
}
