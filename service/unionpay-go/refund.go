package unionpay

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// RefundReq 退款请求参数。
type RefundReq struct {
	OrderID   string // 原商户订单号
	OrigQryID string // 原银联查询流水号
	TxnAmt    int64  // 退款金额（分）
	BackURL   string // 回调地址
	SubMerID  string // 可选
}

// RefundResp 退款响应。
type RefundResp struct {
	QueryID string // 退款流水号
	TxnAmt  int64  // 实际退款金额（分）
}

// RefundService 提供退款相关的 API 方法。
// 对标 wechatpay-go refunddomestic.RefundsApiService。
type RefundService struct{ Service }

// Refund 发起退款（txnType=04 退货）。
func (s *RefundService) Refund(ctx context.Context, req *RefundReq) (*RefundResp, error) {
	txnTime := time.Now().Format("20060102150405")

	params := map[string]string{
		"version":      "5.1.0",
		"encoding":     "UTF-8",
		"signMethod":   "01",
		"txnType":      "04",
		"txnSubType":   "00",
		"bizType":      "000000",
		"accessType":   "0",
		"merId":        s.Client.mchID,
		"orderId":      req.OrderID,
		"txnTime":      txnTime,
		"txnAmt":       strconv.FormatInt(req.TxnAmt, 10),
		"currencyCode": "156",
		"backUrl":      req.BackURL,
	}
	if req.OrigQryID != "" {
		params["origQryId"] = req.OrigQryID
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
		return nil, fmt.Errorf("unionpay-go: refund error: %s (code=%s)", respMsg, respCode)
	}

	refundAmt, _ := strconv.ParseInt(respParams.Get("txnAmt"), 10, 64)

	return &RefundResp{
		QueryID: respParams.Get("queryId"),
		TxnAmt:  refundAmt,
	}, nil
}
