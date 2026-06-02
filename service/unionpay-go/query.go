package unionpay

import (
	"context"
	"fmt"
	"time"
)

// QueryReq 订单查询请求参数。
type QueryReq struct {
	OrderID string // 商户订单号
}

// QueryResp 订单查询响应。
type QueryResp struct {
	QueryID      string // 银联查询流水号
	OrigRespCode string // 原交易响应码 ("00" = 支付成功)
	RespMsg      string
}

// QueryService 提供订单查询相关的 API 方法。
// 对标 wechatpay-go NativeApiService.QueryOrderByOutTradeNo()。
type QueryService struct{ Service }

// QueryOrder 根据商户订单号查询订单状态。
func (s *QueryService) QueryOrder(ctx context.Context, req *QueryReq) (*QueryResp, error) {
	txnTime := time.Now().Format("20060102150405")

	params := map[string]string{
		"version":    "5.1.0",
		"encoding":   "UTF-8",
		"signMethod": "01",
		"txnType":    "00",
		"txnSubType": "00",
		"bizType":    "000000",
		"accessType": "0",
		"merId":      s.Client.mchID,
		"orderId":    req.OrderID,
		"txnTime":    txnTime,
	}

	sig, err := s.Client.sign(params)
	if err != nil {
		return nil, fmt.Errorf("unionpay-go: sign failed: %w", err)
	}
	params["signature"] = sig

	respParams, err := s.Client.doPost(ctx, s.Client.queryTransURL(), params)
	if err != nil {
		return nil, err
	}

	respCode := respParams.Get("respCode")
	respMsg := respParams.Get("respMsg")
	if respCode != "00" {
		return nil, fmt.Errorf("unionpay-go: query error: %s (code=%s)", respMsg, respCode)
	}

	return &QueryResp{
		QueryID:      respParams.Get("queryId"),
		OrigRespCode: respParams.Get("origRespCode"),
		RespMsg:      respMsg,
	}, nil
}
