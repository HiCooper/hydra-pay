package ecny

import (
	"context"
	"fmt"
	"strconv"
)

// ---- Request/Response Types ----

// QueryReq 订单查询请求参数。
type QueryReq struct {
	OrderID     string // 商户订单号
	ChannelTxID string // 受理机构交易流水号（优先使用，如果已知）
}

// QueryResp 订单查询响应。
type QueryResp struct {
	ChannelTxID string // 受理机构交易流水号
	OrderID     string // 商户订单号
	TxnAmt      int64  // 交易金额（分）
	Status      string // 交易状态: SUCCESS, PROCESSING, CLOSED, FAILED
	RespCode    string // 原始响应码
	RespMsg     string // 响应信息
}

// ---- Service ----

// QueryService 提供订单查询相关的 API 方法。
type QueryService struct{ Service }

// QueryOrder 查询订单状态。
func (s *QueryService) QueryOrder(ctx context.Context, req *QueryReq) (*QueryResp, error) {
	params := map[string]string{
		"orderId": req.OrderID,
	}
	if req.ChannelTxID != "" {
		params["channelTxId"] = req.ChannelTxID
	}

	sig, err := s.Client.sign(params)
	if err != nil {
		return nil, fmt.Errorf("ecny-go: sign failed: %w", err)
	}
	params["sign"] = sig

	respBody, err := s.Client.doJSONPost(ctx, s.Client.orderQueryURL(), params)
	if err != nil {
		return nil, err
	}

	respCode, _ := respBody["respCode"].(string)
	respMsg, _ := respBody["respMsg"].(string)
	if respCode != "00" && respCode != "0000" && respCode != "SUCCESS" {
		return nil, fmt.Errorf("ecny-go: query order error: %s (code=%s)", respMsg, respCode)
	}

	channelTxID, _ := respBody["channelTxId"].(string)
	if channelTxID == "" {
		channelTxID, _ = respBody["transactionId"].(string)
	}
	orderID, _ := respBody["orderId"].(string)

	status, _ := respBody["status"].(string)
	if status == "" {
		status, _ = respBody["tradeState"].(string)
	}

	var txnAmt int64
	switch v := respBody["txnAmt"].(type) {
	case float64:
		txnAmt = int64(v)
	case string:
		txnAmt, _ = strconv.ParseInt(v, 10, 64)
	}

	return &QueryResp{
		ChannelTxID: channelTxID,
		OrderID:     orderID,
		TxnAmt:      txnAmt,
		Status:      status,
		RespCode:    respCode,
		RespMsg:     respMsg,
	}, nil
}
