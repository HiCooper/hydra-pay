package admin

import (
	"encoding/json"
	"fmt"
	"strings"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	query.Count(&total)
	query.Order("created_at DESC").Limit(50).Find(&payments)

	response.Success(c, gin.H{
		"orders": payments,
		"total":  total,
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
		c.Writer.WriteString(csvEscape(p.ID.String()) + "," +
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

	var appName string
	var app model.App
	if err := h.db.First(&app, "id = ?", payment.AppID).Error; err == nil {
		appName = app.Name
	}

	response.Success(c, gin.H{
		"payment":  payment,
		"app_name": appName,
		"events":   events,
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

	pid, err := uuid.Parse(req.PaymentID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid payment id")
		return
	}

	payment, err := h.paymentRepo.GetByID(pid)
	if err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "payment not found")
		return
	}

	if payment.Status == model.PaymentStatusPaid {
		response.Success(c, gin.H{"message": "already paid", "payment": payment})
		return
	}

	applied, _ := h.paymentRepo.MarkPaidIfPending(pid, "simulated_tx_"+req.PaymentID[:8])
	if !applied {
		response.Error(c, http.StatusConflict, "CONFLICT", "payment already in terminal state")
		return
	}

	payment.Status = model.PaymentStatusPaid
	repository.RecordEvent(h.db, model.EventCallbackReceived, payment.Channel,
		pid, "simulated callback",
		map[string]interface{}{"status": req.Status, "simulated": true}, "")

	repository.RecordEvent(h.db, model.EventStatusChanged, payment.Channel,
		pid, "",
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

var _ = json.Marshal
