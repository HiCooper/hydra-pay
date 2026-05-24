package admin

import (
	"context"
	"encoding/json"
	"log"
	"fmt"
	"strings"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smartwalle/alipay/v3"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/pkg/response"
)

type Handler struct {
	db          *gorm.DB
	paymentRepo *repository.PaymentRepository
	eventRepo   *repository.EventRepository
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:          db,
		paymentRepo: repository.NewPaymentRepository(db),
		eventRepo:   repository.NewEventRepository(db),
	}
}

// ---- Apps ----

func (h *Handler) ListApps(c *gin.Context) {
	var apps []model.App
	h.db.Order("created_at DESC").Find(&apps)
	response.Success(c, apps)
}

func (h *Handler) GetApp(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid app id")
		return
	}
	var app model.App
	if err := h.db.First(&app, "id = ?", id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "app not found")
		return
	}
	response.Success(c, app)
}

func (h *Handler) CreateApp(c *gin.Context) {
	var req struct {
		Name           string `json:"name"`
		AlipayPID      string `json:"alipay_pid"`
		WechatSubMchid string `json:"wechat_sub_mchid"`
		WechatSubAppid string `json:"wechat_sub_appid"`
		WebhookURL    string `json:"webhook_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.Name == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "name is required")
		return
	}
	app := model.App{
		Name:           req.Name,
		APIKey:         model.GenerateAPIKey(),
		Status:         "active",
		AlipayPID:      req.AlipayPID,
		WechatSubMchid: req.WechatSubMchid,
		WechatSubAppid: req.WechatSubAppid,
		WebhookURL:    req.WebhookURL,
	}
	if err := h.db.Create(&app).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	response.Success(c, app)
}

func (h *Handler) UpdateApp(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid app id")
		return
	}
	var req struct {
		Name           *string `json:"name"`
		Status         *string `json:"status"`
		AlipayPID      *string `json:"alipay_pid"`
		WechatSubMchid *string `json:"wechat_sub_mchid"`
		WechatSubAppid *string `json:"wechat_sub_appid"`
		WebhookURL    *string `json:"webhook_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.AlipayPID != nil {
		updates["alipay_pid"] = *req.AlipayPID
	}
	if req.WechatSubMchid != nil {
		updates["wechat_sub_mchid"] = *req.WechatSubMchid
	}
	if req.WechatSubAppid != nil {
		updates["wechat_sub_appid"] = *req.WechatSubAppid
	}
	if req.WebhookURL != nil {
		updates["webhook_url"] = *req.WebhookURL
	}
	if err := h.db.Model(&model.App{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	var app model.App
	h.db.First(&app, "id = ?", id)
	response.Success(c, app)
}

// ---- Orders ----

func (h *Handler) ListOrders(c *gin.Context) {
	var payments []model.Payment
	var total int64

	query := h.db.Model(&model.Payment{})

	if appID := c.Query("app_id"); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if channel := c.Query("channel"); channel != "" {
		query = query.Where("channel = ?", channel)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if tradeNo := c.Query("trade_no"); tradeNo != "" {
		query = query.Where("trade_no LIKE ?", "%"+tradeNo+"%")
	}

	page := 1
	pageSize := 10
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&payments)

	response.Success(c, gin.H{
		"orders":    payments,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *Handler) ExportOrders(c *gin.Context) {
	var payments []model.Payment
	query := h.db.Model(&model.Payment{})
	if appID := c.Query("app_id"); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if channel := c.Query("channel"); channel != "" {
		query = query.Where("channel = ?", channel)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	query.Order("created_at DESC").Limit(5000).Find(&payments)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=orders.csv")
	// BOM for Excel UTF-8 compatibility
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	c.Writer.WriteString("订单ID,渠道,金额(元),币种,状态,外部交易号,描述,创建时间,支付时间\n")
	for _, p := range payments {
		paidAt := ""
		if p.PaidAt != nil {
			paidAt = p.PaidAt.Format("2006-01-02 15:04:05")
		}
		c.Writer.WriteString(csvEscape(p.TradeNo) + "," +
			p.Channel + "," +
			fmt.Sprintf("%.2f", float64(p.Amount)/100) + "," +
			p.Currency + "," +
			p.Status + "," +
			csvEscape(p.ExternalID) + "," +
			csvEscape(p.Description) + "," +
			p.CreatedAt.Format("2006-01-02 15:04:05") + "," +
			paidAt + "\n")
	}
}

func csvEscape(s string) string {
	if s == "" {
		return ""
	}
	return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
}

func (h *Handler) GetOrder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid payment id")
		return
	}

	payment, err := h.paymentRepo.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "payment not found")
		return
	}

	events, _ := h.eventRepo.ListByPayment(id)

	var alipayCbs []model.AlipayCallback
	h.db.Where("payment_id = ?", id).Order("created_at DESC").Find(&alipayCbs)

	var wechatCbs []model.WeChatCallback
	h.db.Where("payment_id = ?", id).Order("created_at DESC").Find(&wechatCbs)

	var refunds []model.Refund
	h.db.Where("payment_id = ?", id).Order("created_at DESC").Find(&refunds)

	var appName string
	var app model.App
	if err := h.db.First(&app, "id = ?", payment.AppID).Error; err == nil {
		appName = app.Name
	}

	response.Success(c, gin.H{
		"payment":         payment,
		"app_name":        appName,
		"events":          events,
		"refunds":          refunds,
		"alipay_callbacks": alipayCbs,
		"wechat_callbacks": wechatCbs,
	})
}

// ---- Events ----

func (h *Handler) ListEvents(c *gin.Context) {
	var events []model.PaymentEvent
	query := h.db.Model(&model.PaymentEvent{})

	if paymentID := c.Query("payment_id"); paymentID != "" {
		query = query.Where("payment_id = ?", paymentID)
	}

	query.Order("created_at DESC").Limit(100).Find(&events)
	response.Success(c, events)
}

// ---- Dashboard ----

func (h *Handler) Dashboard(c *gin.Context) {
	today := time.Now().Truncate(24 * time.Hour)

	var todayOrders, todayPaid int64
	h.db.Model(&model.Payment{}).Where("created_at >= ?", today).Count(&todayOrders)
	h.db.Model(&model.Payment{}).Where("created_at >= ? AND status = ?", today, model.PaymentStatusPaid).Count(&todayPaid)

	var todayRevenue float64
	h.db.Model(&model.Payment{}).
		Where("created_at >= ? AND status = ?", today, model.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&todayRevenue)

	type chanStat struct {
		Channel string
		Count   int64
	}
	var channelStats []chanStat
	h.db.Model(&model.Payment{}).
		Where("created_at >= ?", today).
		Select("channel, COUNT(*) as count").
		Group("channel").Find(&channelStats)

	successRate := float64(0)
	if todayOrders > 0 {
		successRate = float64(todayPaid) / float64(todayOrders) * 100
	}

	response.Success(c, gin.H{
		"today_orders":  todayOrders,
		"today_paid":    todayPaid,
		"today_revenue": todayRevenue / 100,
		"success_rate":  successRate,
		"channel_stats": channelStats,
	})
}

// ---- Config ----

func (h *Handler) ChannelConfig(c *gin.Context) {
	response.Success(c, gin.H{
		"alipay": gin.H{
			"app_id":     maskString(os.Getenv("ALIPAY_APP_ID"), 4),
			"sandbox":    os.Getenv("ALIPAY_SANDBOX"),
			"key_loaded": os.Getenv("ALIPAY_PRIVATE_KEY") != "" || os.Getenv("ALIPAY_PRIVATE_KEY_PATH") != "",
			"pub_loaded": os.Getenv("ALIPAY_ALIPAY_PUBLIC_KEY") != "" || os.Getenv("ALIPAY_ALIPAY_PUBLIC_KEY_PATH") != "",
			"notify_url": os.Getenv("ALIPAY_NOTIFY_URL"),
			"return_url": os.Getenv("ALIPAY_RETURN_URL"),
		},
		"wechat": gin.H{
			"mch_id":     maskString(os.Getenv("WECHAT_MCH_ID"), 4),
			"serial_no":  maskString(os.Getenv("WECHAT_SERIAL_NO"), 4),
			"key_loaded": os.Getenv("WECHAT_PRIVATE_KEY") != "" || os.Getenv("WECHAT_PRIVATE_KEY_PATH") != "",
			"notify_url": os.Getenv("WECHAT_NOTIFY_URL"),
		},
		"global_webhook": os.Getenv("WALL_WEBHOOK_URL"),
	})
}

func maskString(s string, show int) string {
	if s == "" {
		return "(未配置)"
	}
	if len(s) <= show {
		return s
	}
	return s[:show] + "****"
}

// ---- Tools ----

func (h *Handler) SimulateCallback(c *gin.Context) {
	var req struct {
		PaymentID string `json:"payment_id"`
		Status    string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.Status == "" {
		req.Status = model.PaymentStatusPaid
	}

	var payment *model.Payment
	pid, err := uuid.Parse(req.PaymentID)
	if err == nil {
		payment, err = h.paymentRepo.GetByID(pid)
	} else {
		payment, err = h.paymentRepo.GetByTradeNo(req.PaymentID)
	}
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "payment not found")
		return
	}

	if payment.Status == model.PaymentStatusPaid {
		response.Success(c, gin.H{"message": "already paid", "payment": payment})
		return
	}

	applied, _ := h.paymentRepo.MarkPaidIfPending(payment.ID, "simulated_tx_"+req.PaymentID[:8])
	if !applied {
		response.Error(c, http.StatusConflict, "CONFLICT", "payment already in terminal state")
		return
	}

	payment.Status = model.PaymentStatusPaid
	repository.RecordEvent(h.db, model.EventCallbackReceived, payment.Channel,
		payment.ID, "simulated callback",
		map[string]interface{}{"status": req.Status, "simulated": true}, "")

	repository.RecordEvent(h.db, model.EventStatusChanged, payment.Channel,
		payment.ID, "",
		map[string]interface{}{"from": "processing", "to": "paid"}, "")

	response.Success(c, gin.H{"message": "callback simulated", "payment": payment})
}

func (h *Handler) TestWebhook(c *gin.Context) {
	var req struct {
		AppID string `json:"app_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	aid, err := uuid.Parse(req.AppID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid app id")
		return
	}

	var app model.App
	if err := h.db.First(&app, "id = ?", aid).Error; err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "app not found")
		return
	}
	if app.WebhookURL == "" {
		response.Error(c, http.StatusBadRequest, "NO_WEBHOOK", "app has no webhook_url configured")
		return
	}

	payload := fmt.Sprintf(`{"event":"test","app_id":"%s","timestamp":"%s","message":"this is a test webhook from 星河支付"}`, app.ID, time.Now().Format(time.RFC3339))

	resp, err := http.Post(app.WebhookURL, "application/json", strings.NewReader(payload))
	if err != nil {
		response.Error(c, http.StatusBadGateway, "SEND_FAILED", fmt.Sprintf("webhook send failed: %v", err))
		return
	}
	defer resp.Body.Close()

	response.Success(c, gin.H{
		"message":       "webhook sent",
		"webhook_url":   app.WebhookURL,
		"response_code": resp.StatusCode,
	})
}

func (h *Handler) ConnectivityCheck(c *gin.Context) {
	type result struct {
		Channel string `json:"channel"`
		Gateway string `json:"gateway"`
		Status  string `json:"status"`
		Latency string `json:"latency"`
	}

	results := []result{}

	// Alipay sandbox
	start := time.Now()
	resp, err := http.Get("https://openapi-sandbox.dl.alipaydev.com/gateway.do")
	lat := time.Since(start)
	r := result{Channel: "alipay", Gateway: "openapi-sandbox.dl.alipaydev.com"}
	if err != nil {
		r.Status = "unreachable"
		r.Latency = lat.String()
	} else {
		resp.Body.Close()
		r.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
		r.Latency = lat.String()
	}
	results = append(results, r)

	// Alipay production
	start = time.Now()
	resp2, err2 := http.Get("https://openapi.alipay.com/gateway.do")
	lat2 := time.Since(start)
	r2 := result{Channel: "alipay", Gateway: "openapi.alipay.com"}
	if err2 != nil {
		r2.Status = "unreachable"
		r2.Latency = lat2.String()
	} else {
		resp2.Body.Close()
		r2.Status = fmt.Sprintf("HTTP %d", resp2.StatusCode)
		r2.Latency = lat2.String()
	}
	results = append(results, r2)

	// WeChat Pay
	start = time.Now()
	resp3, err3 := http.Get("https://api.mch.weixin.qq.com")
	lat3 := time.Since(start)
	r3 := result{Channel: "wechat", Gateway: "api.mch.weixin.qq.com"}
	if err3 != nil {
		r3.Status = "unreachable"
		r3.Latency = lat3.String()
	} else {
		resp3.Body.Close()
		r3.Status = fmt.Sprintf("HTTP %d", resp3.StatusCode)
		r3.Latency = lat3.String()
	}
	results = append(results, r3)

	response.Success(c, gin.H{"results": results})
}

func (h *Handler) TestRefund(c *gin.Context) {
	var req struct {
		TradeNo      string `json:"trade_no"`
		RefundAmount string `json:"refund_amount"`
		RefundReason string `json:"refund_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.TradeNo == "" || req.RefundAmount == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "trade_no and refund_amount are required")
		return
	}

	payment, err := h.paymentRepo.GetByTradeNo(req.TradeNo)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "payment not found")
		return
	}
	if payment.Status != model.PaymentStatusPaid {
		response.Error(c, http.StatusBadRequest, "NOT_PAID", "only paid orders can be refunded")
		return
	}

	outReqNo := "RF" + payment.TradeNo
	ctx := c.Request.Context()

	switch payment.Channel {
	case model.ChannelAlipay:
		alipayRefund(c, payment, req.RefundAmount, req.RefundReason, outReqNo, ctx, h)
	case model.ChannelWechat:
		wechatRefund(c, payment, req.RefundAmount, req.RefundReason, outReqNo, ctx, h)
	default:
		response.Error(c, http.StatusBadRequest, "UNSUPPORTED", "unsupported channel: "+payment.Channel)
	}
}

func alipayRefund(c *gin.Context, payment *model.Payment, amount, reason, outReqNo string, ctx context.Context, h *Handler) {
	appID := os.Getenv("ALIPAY_APP_ID")
	privateKey := resolveAdminKey()
	if appID == "" || privateKey == "" {
		response.Error(c, http.StatusInternalServerError, "CONFIG_ERROR", "alipay not configured")
		return
	}

	isSandbox := os.Getenv("ALIPAY_SANDBOX") == "true"
	client, err := alipay.New(appID, privateKey, !isSandbox)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "CLIENT_ERROR", "failed to create alipay client: "+err.Error())
		return
	}

	pubKey := os.Getenv("ALIPAY_ALIPAY_PUBLIC_KEY")
	if pubKey != "" {
		if err := client.LoadAliPayPublicKey(pubKey); err != nil {
			response.Error(c, http.StatusInternalServerError, "CLIENT_ERROR", "failed to load alipay public key: "+err.Error())
			return
		}
	}

	p := alipay.TradeRefund{
		OutTradeNo:   payment.TradeNo,
		RefundAmount: amount,
		OutRequestNo: outReqNo,
	}
	if reason != "" {
		p.RefundReason = reason
	}

	resp, err := client.TradeRefund(ctx, p)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "REFUND_FAILED", err.Error())
		return
	}
	if resp.Code != alipay.CodeSuccess {
		response.Error(c, http.StatusBadGateway, "REFUND_FAILED",
			fmt.Sprintf("%s (code=%s)", resp.Msg, resp.Code))
		return
	}

	if err := h.paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusRefunded, payment.ExternalID); err != nil {
		log.Printf("[refund] update status failed: %v", err)
	}

	respJSON, _ := json.Marshal(resp)
	h.db.Create(&model.Refund{
		PaymentID:       payment.ID,
		TradeNo:         payment.TradeNo,
		Channel:         model.ChannelAlipay,
		RefundAmount:    amount,
		RefundReason:    reason,
		OutRequestNo:    outReqNo,
		Status:          model.RefundStatusSuccess,
		ChannelRefundID: resp.TradeNo,
		ChannelTxID:     resp.TradeNo,
		RefundFee:       resp.RefundFee,
		ResponseData:    respJSON,
	})
	repository.RecordEvent(h.db, model.EventRefund, model.ChannelAlipay,
		payment.ID, string(respJSON),
		map[string]interface{}{"refund_fee": resp.RefundFee, "trade_no": resp.TradeNo}, "")

	response.Success(c, gin.H{
		"message":     "refund success",
		"channel":     "alipay",
		"trade_no":    resp.TradeNo,
		"refund_fee":  resp.RefundFee,
		"fund_change": resp.FundChange,
	})
}

func wechatRefund(c *gin.Context, payment *model.Payment, amount, reason, outReqNo string, ctx context.Context, h *Handler) {
	mchID := os.Getenv("WECHAT_MCH_ID")
	serialNo := os.Getenv("WECHAT_SERIAL_NO")
	apiV3Key := os.Getenv("WECHAT_API_V3_KEY")
	privateKey := resolveWechatAdminKey()
	if mchID == "" || apiV3Key == "" || serialNo == "" || privateKey == "" {
		response.Error(c, http.StatusInternalServerError, "CONFIG_ERROR", "wechat not configured")
		return
	}

	priv, err := utils.LoadPrivateKey(privateKey)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "CLIENT_ERROR", "failed to load wechat private key: "+err.Error())
		return
	}

	client, err := core.NewClient(ctx,
		option.WithWechatPayAutoAuthCipher(mchID, serialNo, priv, apiV3Key),
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "CLIENT_ERROR", "failed to create wechat client: "+err.Error())
		return
	}

	// amount is in yuan string, wechat needs cents
	amountF, _ := strconv.ParseFloat(amount, 64)
	amountCents := int64(amountF * 100)

	svc := refunddomestic.RefundsApiService{Client: client}
	req := refunddomestic.CreateRequest{
		OutTradeNo:  core.String(payment.TradeNo),
		OutRefundNo: core.String(outReqNo),
		Amount:      &refunddomestic.AmountReq{Refund: core.Int64(amountCents), Total: core.Int64(payment.Amount), Currency: core.String("CNY")},
	}
	if reason != "" {
		req.Reason = core.String(reason)
	}

	resp, _, err := svc.Create(ctx, req)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "REFUND_FAILED", err.Error())
		return
	}

	if err := h.paymentRepo.UpdateStatus(payment.ID, model.PaymentStatusRefunded, payment.ExternalID); err != nil {
		log.Printf("[refund] update status failed: %v", err)
	}

	respJSON, _ := json.Marshal(resp)
	h.db.Create(&model.Refund{
		PaymentID:       payment.ID,
		TradeNo:         payment.TradeNo,
		Channel:         model.ChannelWechat,
		RefundAmount:    amount,
		RefundReason:    reason,
		OutRequestNo:    outReqNo,
		Status:          model.RefundStatusSuccess,
		ChannelRefundID: *resp.RefundId,
		ChannelTxID:     *resp.TransactionId,
		RefundFee:       fmt.Sprintf("%.2f", float64(amountCents)/100),
		ResponseData:    respJSON,
	})
	repository.RecordEvent(h.db, model.EventRefund, model.ChannelWechat,
		payment.ID, string(respJSON),
		map[string]interface{}{"refund_id": *resp.RefundId, "status": string(*resp.Status)}, "")

	response.Success(c, gin.H{
		"message":        "refund success",
		"channel":        "wechat",
		"refund_id":      *resp.RefundId,
		"out_refund_no":  *resp.OutRefundNo,
		"transaction_id": *resp.TransactionId,
		"status":         string(*resp.Status),
	})
}

func resolveAdminKey() string {
	if k := os.Getenv("ALIPAY_PRIVATE_KEY"); k != "" {
		return k
	}
	if p := os.Getenv("ALIPAY_PRIVATE_KEY_PATH"); p != "" {
		data, _ := os.ReadFile(p)
		return string(data)
	}
	return ""
}

func resolveWechatAdminKey() string {
	if k := os.Getenv("WECHAT_PRIVATE_KEY"); k != "" {
		return k
	}
	if p := os.Getenv("WECHAT_PRIVATE_KEY_PATH"); p != "" {
		data, _ := os.ReadFile(p)
		return string(data)
	}
	return ""
}

var _ = json.Marshal
