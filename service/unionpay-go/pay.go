package unionpay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ---- Request/Response Types ----

// QRCodePayReq 扫码支付（Native）请求参数。
type QRCodePayReq struct {
	OrderID   string // 商户订单号
	TxnAmt    int64  // 交易金额（分）
	BackURL   string // 后台回调地址
	SubMerID  string // 可选：间联子商户号
}

// QRCodePayResp 扫码支付响应。
type QRCodePayResp struct {
	QueryID  string // 银联查询流水号
	QRCode   string // 二维码链接
	RespCode string // 响应码 ("00" = 成功)
	RespMsg  string // 响应信息
}

// AppPayReq App 支付请求参数。
type AppPayReq struct {
	OrderID  string // 商户订单号
	TxnAmt   int64  // 交易金额（分）
	BackURL  string // 后台回调地址
	SubMerID string // 可选
}

// AppPayResp App 支付响应。
type AppPayResp struct {
	QueryID string // 银联查询流水号
	TN      string // 交易流水号（供 UPSDK 调起）
}

// H5PayReq H5/WAP 支付请求参数。
type H5PayReq struct {
	OrderID   string
	TxnAmt    int64
	BackURL   string
	FrontURL  string // H5 前端返回地址
	SubMerID  string
}

// ---- Service ----

// PayService 提供支付下单（QRCode/App/H5）相关的 API 方法。
// 对标 wechatpay-go services/payments/native.NativeApiService。
type PayService struct{ Service }

// QRCodePay 扫码支付下单，返回二维码链接供用户扫码。
// 对标 wechatpay-go NativeApiService.Prepay()。
func (s *PayService) QRCodePay(ctx context.Context, req *QRCodePayReq) (*QRCodePayResp, error) {
	txnTime := time.Now().Format("20060102150405")

	params := map[string]string{
		"version":      "5.1.0",
		"encoding":     "UTF-8",
		"signMethod":   "01",
		"txnType":      "01",
		"txnSubType":   "07",
		"bizType":      "000201",
		"channelType":  "08",
		"accessType":   "0",
		"merId":        s.Client.mchID,
		"orderId":      req.OrderID,
		"txnTime":      txnTime,
		"txnAmt":       strconv.FormatInt(req.TxnAmt, 10),
		"currencyCode": "156",
		"backUrl":      req.BackURL,
	}
	if req.SubMerID != "" {
		params["subMerId"] = req.SubMerID
	}

	sig, err := s.Client.sign(params)
	if err != nil {
		return nil, fmt.Errorf("unionpay-go: sign failed: %w", err)
	}
	params["signature"] = sig

	respParams, err := s.Client.doPost(ctx, s.Client.backTransURL(), params)
	if err != nil {
		return nil, err
	}

	respCode := respParams.Get("respCode")
	respMsg := respParams.Get("respMsg")
	if respCode != "00" {
		return nil, fmt.Errorf("unionpay-go: qrcode pay error: %s (code=%s)", respMsg, respCode)
	}

	return &QRCodePayResp{
		QueryID:  respParams.Get("queryId"),
		QRCode:   respParams.Get("qrCode"),
		RespCode: respCode,
		RespMsg:  respMsg,
	}, nil
}

// AppPay App 支付下单，返回 TN 供客户端 UPSDK 调起支付。
func (s *PayService) AppPay(ctx context.Context, req *AppPayReq) (*AppPayResp, error) {
	txnTime := time.Now().Format("20060102150405")

	params := map[string]string{
		"version":      "5.1.0",
		"encoding":     "UTF-8",
		"signMethod":   "01",
		"txnType":      "01",
		"txnSubType":   "01",
		"bizType":      "000000",
		"channelType":  "08",
		"accessType":   "0",
		"merId":        s.Client.mchID,
		"orderId":      req.OrderID,
		"txnTime":      txnTime,
		"txnAmt":       strconv.FormatInt(req.TxnAmt, 10),
		"currencyCode": "156",
		"backUrl":      req.BackURL,
	}
	if req.SubMerID != "" {
		params["subMerId"] = req.SubMerID
	}

	sig, err := s.Client.sign(params)
	if err != nil {
		return nil, fmt.Errorf("unionpay-go: sign failed: %w", err)
	}
	params["signature"] = sig

	respParams, err := s.Client.doPost(ctx, s.Client.appTransURL(), params)
	if err != nil {
		return nil, err
	}

	respCode := respParams.Get("respCode")
	respMsg := respParams.Get("respMsg")
	if respCode != "00" {
		return nil, fmt.Errorf("unionpay-go: app pay error: %s (code=%s)", respMsg, respCode)
	}

	return &AppPayResp{
		QueryID: respParams.Get("queryId"),
		TN:      respParams.Get("tn"),
	}, nil
}

// H5Pay H5/WAP 支付下单，返回 HTML 自动提交表单。
func (s *PayService) H5Pay(ctx context.Context, req *H5PayReq) (string, error) {
	txnTime := time.Now().Format("20060102150405")

	params := map[string]string{
		"version":      "5.1.0",
		"encoding":     "UTF-8",
		"signMethod":   "01",
		"txnType":      "01",
		"txnSubType":   "01",
		"bizType":      "000201",
		"channelType":  "08",
		"accessType":   "0",
		"merId":        s.Client.mchID,
		"orderId":      req.OrderID,
		"txnTime":      txnTime,
		"txnAmt":       strconv.FormatInt(req.TxnAmt, 10),
		"currencyCode": "156",
		"backUrl":      req.BackURL,
		"frontUrl":     req.FrontURL,
	}
	if req.SubMerID != "" {
		params["subMerId"] = req.SubMerID
	}

	sig, err := s.Client.sign(params)
	if err != nil {
		return "", fmt.Errorf("unionpay-go: sign failed: %w", err)
	}
	params["signature"] = sig

	var fields bytes.Buffer
	for k, v := range params {
		fields.WriteString(fmt.Sprintf(
			`<input type="hidden" name="%s" value="%s" />`, k, v))
	}

	return fmt.Sprintf(`<html><body onload="document.forms[0].submit()">
<form action="%s" method="POST">%s</form>
</body></html>`, s.Client.frontTransURL(), fields.String()), nil
}

// ---- Internal HTTP ----

// doPost 向银联网关发送表单 POST 请求，返回解析后的参数。
func (c *Client) doPost(ctx context.Context, reqURL string, params map[string]string) (url.Values, error) {
	form := make(url.Values)
	for k, v := range params {
		if v != "" {
			form.Set(k, v)
		}
	}

	body := form.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unionpay-go: http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unionpay-go: read response failed: %w", err)
	}

	rawBody := string(respBytes)
	parsed, err := url.ParseQuery(rawBody)
	if err != nil {
		return nil, fmt.Errorf("unionpay-go: parse response failed: %w", err)
	}

	// 银联网关在商户号无效时返回纯文本 "Invalid request."，不是 form-encoded
	if len(parsed) == 0 && rawBody != "" {
		return nil, fmt.Errorf("unionpay-go: gateway returned non-form-encoded response (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(rawBody))
	}

	return parsed, nil
}
