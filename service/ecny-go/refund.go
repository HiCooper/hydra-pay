package ecny

import (
	"context"
	"fmt"
	"strconv"
)

// ---- Request/Response Types ----

// RefundReq 退款请求参数。
type RefundReq struct {
	OrderID      string // 商户订单号
	ChannelTxID  string // 原交易受理机构流水号
	RefundAmount int64  // 退款金额（分）
	RefundReason string // 退款原因
	BackURL      string // 退款结果异步通知地址
	SubMerID     string // 可选：间联子商户号
}

// RefundResp 退款响应。
type RefundResp struct {
	ChannelRefundID string // 受理机构退款流水号
	RefundAmount    int64  // 实际退款金额（分）
	RespCode        string
	RespMsg         string
}

// ---- Service ----

// RefundService 提供退款相关的 API 方法。
type RefundService struct{ Service }

// Refund 发起退款。
func (s *RefundService) Refund(ctx context.Context, req *RefundReq) (*RefundResp, error) {
	params := map[string]string{
		"orderId":      req.OrderID,
		"channelTxId":  req.ChannelTxID,
		"refundAmount": strconv.FormatInt(req.RefundAmount, 10),
		"backUrl":      req.BackURL,
	}
	if req.RefundReason != "" {
		params["refundReason"] = req.RefundReason
	}
	if req.SubMerID != "" {
		params["subMerId"] = req.SubMerID
	}

	sig, err := s.Client.sign(params)
	if err != nil {
		return nil, fmt.Errorf("ecny-go: sign failed: %w", err)
	}
	params["sign"] = sig

	respBody, err := s.Client.doJSONPost(ctx, s.Client.orderRefundURL(), params)
	if err != nil {
		return nil, err
	}

	respCode, _ := respBody["respCode"].(string)
	respMsg, _ := respBody["respMsg"].(string)
	if respCode != "00" && respCode != "0000" && respCode != "SUCCESS" {
		return nil, fmt.Errorf("ecny-go: refund error: %s (code=%s)", respMsg, respCode)
	}

	channelRefundID, _ := respBody["refundId"].(string)
	if channelRefundID == "" {
		channelRefundID, _ = respBody["channelRefundId"].(string)
	}

	var refundAmt int64
	switch v := respBody["refundAmount"].(type) {
	case float64:
		refundAmt = int64(v)
	case string:
		refundAmt, _ = strconv.ParseInt(v, 10, 64)
	}

	return &RefundResp{
		ChannelRefundID: channelRefundID,
		RefundAmount:    refundAmt,
		RespCode:        respCode,
		RespMsg:         respMsg,
	}, nil
}
