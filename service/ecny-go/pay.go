package ecny

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// ---- Request/Response Types ----

// QRCodePayReq 扫码支付（Native，主扫）请求参数。
// 商户生成二维码，用户在数字人民币 APP 中扫码支付。
type QRCodePayReq struct {
	OrderID  string // 商户订单号（TradeNo）
	TxnAmt   int64  // 交易金额（分）
	BackURL  string // 后台异步通知地址
	Subject  string // 商品描述
	SubMerID string // 可选：间联子商户号
}

// QRCodePayResp 扫码支付响应。
type QRCodePayResp struct {
	ChannelTxID string // 受理机构交易流水号（幂等键）
	QRCodeURL   string // 二维码链接
	RespCode    string // 响应码
	RespMsg     string // 响应信息
}

// AppPayReq App 拉起支付请求参数。
// 返回加密订单信息，客户端用其唤起数字人民币收银台（SDK 或 URL Scheme）。
// 对标瑞幸咖啡的「拉起支付」模式。
type AppPayReq struct {
	OrderID      string // 商户订单号
	TxnAmt       int64  // 交易金额（分）
	BackURL      string // 后台异步通知地址
	Subject      string // 商品描述
	SubMerID     string // 可选：间联子商户号
	LastWalletID string // 可选：用户上次使用的钱包 ID（用于钱包推荐）
}

// AppPayResp App 拉起支付响应。
// EncryptedInfo/EncryptedKey 等字段为加密后的订单核心信息，
// 客户端透传给数字人民币收银台 SDK，无法解密的密文保证了金额不可篡改。
type AppPayResp struct {
	ChannelTxID string // 受理机构交易流水号
	// ---- 加密订单信息（客户端透传给收银台 SDK） ----
	EncryptedKey  string // SM2/SM4 加密后的对称密钥密文
	EncryptedInfo string // SM4 加密后的订单核心信息
	EncryptionSN  string // 加密证书序列号/密钥标识
	ExtraInfo     string // 扩展信息
	// ----
	RespCode string
	RespMsg  string
}

// ---- Service ----

// PayService 提供支付下单（QRCode/App）相关的 API 方法。
// 对标 unionpay-go PayService。
type PayService struct{ Service }

// QRCodePay 扫码支付下单，返回二维码链接供用户扫码。
func (s *PayService) QRCodePay(ctx context.Context, req *QRCodePayReq) (*QRCodePayResp, error) {
	txnTime := time.Now().Format("20060102150405")

	params := map[string]string{
		"orderId":   req.OrderID,
		"txnAmt":    strconv.FormatInt(req.TxnAmt, 10),
		"txnTime":   txnTime,
		"backUrl":   req.BackURL,
		"tradeType": TradeNative,
	}
	if req.Subject != "" {
		params["subject"] = req.Subject
	}
	if req.SubMerID != "" {
		params["subMerId"] = req.SubMerID
	}

	sig, err := s.Client.sign(params)
	if err != nil {
		return nil, fmt.Errorf("ecny-go: sign failed: %w", err)
	}
	params["sign"] = sig

	respBody, err := s.Client.doJSONPost(ctx, s.Client.orderCreateURL(), params)
	if err != nil {
		return nil, err
	}

	respCode, _ := respBody["respCode"].(string)
	respMsg, _ := respBody["respMsg"].(string)
	if respCode != "00" && respCode != "0000" && respCode != "SUCCESS" {
		return nil, fmt.Errorf("ecny-go: qrcode pay error: %s (code=%s)", respMsg, respCode)
	}

	channelTxID, _ := respBody["channelTxId"].(string)
	if channelTxID == "" {
		channelTxID, _ = respBody["transactionId"].(string)
	}
	qrCode, _ := respBody["qrCode"].(string)
	if qrCode == "" {
		qrCode, _ = respBody["qrCodeUrl"].(string)
	}

	return &QRCodePayResp{
		ChannelTxID: channelTxID,
		QRCodeURL:   qrCode,
		RespCode:    respCode,
		RespMsg:     respMsg,
	}, nil
}

// AppPay App 拉起支付下单，返回加密订单信息供客户端唤起数字人民币收银台。
// 对标瑞幸咖啡的「拉起支付」模式——客户端用 EncryptedInfo 等字段调用
// 数字人民币收银台 SDK（或 URL Scheme 跳转数字人民币 APP）完成支付。
func (s *PayService) AppPay(ctx context.Context, req *AppPayReq) (*AppPayResp, error) {
	txnTime := time.Now().Format("20060102150405")

	params := map[string]string{
		"orderId":   req.OrderID,
		"txnAmt":    strconv.FormatInt(req.TxnAmt, 10),
		"txnTime":   txnTime,
		"backUrl":   req.BackURL,
		"tradeType": TradeApp,
		"appId":     s.Client.appID,
	}
	if req.Subject != "" {
		params["subject"] = req.Subject
	}
	if req.SubMerID != "" {
		params["subMerId"] = req.SubMerID
	}
	if req.LastWalletID != "" {
		params["lastWalletId"] = req.LastWalletID
	}

	sig, err := s.Client.sign(params)
	if err != nil {
		return nil, fmt.Errorf("ecny-go: sign failed: %w", err)
	}
	params["sign"] = sig

	respBody, err := s.Client.doJSONPost(ctx, s.Client.orderCreateURL(), params)
	if err != nil {
		return nil, err
	}

	respCode, _ := respBody["respCode"].(string)
	respMsg, _ := respBody["respMsg"].(string)
	if respCode != "00" && respCode != "0000" && respCode != "SUCCESS" {
		return nil, fmt.Errorf("ecny-go: app pay error: %s (code=%s)", respMsg, respCode)
	}

	channelTxID, _ := respBody["channelTxId"].(string)
	if channelTxID == "" {
		channelTxID, _ = respBody["transactionId"].(string)
	}

	return &AppPayResp{
		ChannelTxID:  channelTxID,
		EncryptedKey:  stringField(respBody, "encryptedKey"),
		EncryptedInfo: stringField(respBody, "encryptedInfo"),
		EncryptionSN:  stringField(respBody, "encryptionSN"),
		ExtraInfo:     stringField(respBody, "extraInfo"),
		RespCode:      respCode,
		RespMsg:       respMsg,
	}, nil
}

func stringField(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

// ---- Internal HTTP ----

// doJSONPost 向受理机构网关发送 JSON POST 请求，返回解析后的 JSON 对象。
// 对标 unionpay-go doPost()，但使用 JSON 格式代替 form-encoded。
func (c *Client) doJSONPost(ctx context.Context, reqURL string, params map[string]string) (map[string]interface{}, error) {
	bodyJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("ecny-go: marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ecny-go: http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ecny-go: read response failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ecny-go: http %d: %s", resp.StatusCode, string(respBytes))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("ecny-go: parse response failed: %w (body=%s)", err, string(respBytes))
	}
	return result, nil
}
