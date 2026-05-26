package admin

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/internal/service"
	"github.com/hydra/pay-service/pkg/response"
)

type Handler struct {
	db             *gorm.DB
	paymentRepo    *repository.PaymentRepository
	eventRepo      *repository.EventRepository
	payService     *service.PaymentService
	planRepo       *repository.SubscriptionPlanRepository
	onboardingRepo *repository.OnboardingRepository
}

func NewHandler(db *gorm.DB, payService *service.PaymentService) *Handler {
	return &Handler{
		db:             db,
		paymentRepo:    repository.NewPaymentRepository(db),
		eventRepo:      repository.NewEventRepository(db),
		payService:     payService,
		planRepo:       repository.NewSubscriptionPlanRepository(db),
		onboardingRepo: repository.NewOnboardingRepository(db),
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
		MerchantID string `json:"merchant_id"`
		Name       string `json:"name"`
		WebhookURL string `json:"webhook_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.Name == "" || req.MerchantID == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "name and merchant_id are required")
		return
	}
	mid, err := uuid.Parse(req.MerchantID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid merchant_id")
		return
	}
	// Verify merchant exists
	var merchant model.Merchant
	if err := h.db.First(&merchant, "id = ?", mid).Error; err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "merchant not found")
		return
	}
	app := model.App{
		MerchantID: mid,
		Name:       req.Name,
		APIKey:     model.GenerateAPIKey(),
		Status:     "active",
		WebhookURL: req.WebhookURL,
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
		Name       *string `json:"name"`
		Status     *string `json:"status"`
		WebhookURL *string `json:"webhook_url"`
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

// ---- Onboarding (read-only for admin) ----

func (h *Handler) GetOnboardingStatus(c *gin.Context) {
	merchantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid merchant id")
		return
	}

	records, err := h.onboardingRepo.GetByMerchantID(merchantID)
	if err != nil || len(records) == 0 {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "no onboarding record found")
		return
	}

	ob := &records[0]

	// If not terminal, refresh from channel
	if ob.Status != model.OnboardingStatusApproved && ob.Status != model.OnboardingStatusRejected {
		adapter, err := service.GetAdapter(ob.Channel, h.payService.GetConfig())
		if err == nil {
			if provider, ok := adapter.(channel.OnboardingProvider); ok {
				statusResp, err := provider.QueryOnboarding(c.Request.Context(), ob.ApplymentID)
				if err == nil {
					h.onboardingRepo.UpdateStatus(ob.ID, statusResp.Status)
					ob.Status = statusResp.Status

					if statusResp.SignURL != "" {
						h.onboardingRepo.UpdateSignURL(ob.ID, statusResp.SignURL, statusResp.QRCodeURL)
						ob.SignURL = statusResp.SignURL
						ob.QrCodeURL = statusResp.QRCodeURL
					}

					if statusResp.Status == model.OnboardingStatusApproved && statusResp.SubMerchantID != "" {
						h.onboardingRepo.MarkApproved(ob.ID, statusResp.SubMerchantID)
						ob.Status = model.OnboardingStatusApproved
						ob.SubMerchantID = statusResp.SubMerchantID
						h.autoUpdateMerchant(merchantID, ob.Channel, statusResp.SubMerchantID)
					}
				}
			}
		}
	}

	response.Success(c, ob)
}

func (h *Handler) ListOnboardings(c *gin.Context) {
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

	records, total, err := h.onboardingRepo.List(
		c.Query("merchant_id"),
		c.Query("channel"),
		c.Query("status"),
		page, pageSize,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	response.Success(c, gin.H{
		"onboardings": records,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
	})
}

func (h *Handler) autoUpdateMerchant(merchantID uuid.UUID, channel, subMerchantID string) {
	if subMerchantID == "" {
		return
	}
	var field string
	switch channel {
	case model.ChannelAlipay:
		field = "alipay_pid"
	case model.ChannelWechat:
		field = "wechat_sub_mchid"
	default:
		return
	}
	h.db.Model(&model.Merchant{}).Where("id = ?", merchantID).Update(field, subMerchantID)
}

// ---- Merchants ----

func (h *Handler) ListMerchants(c *gin.Context) {
	var merchants []model.Merchant
	h.db.Order("created_at DESC").Find(&merchants)
	response.Success(c, merchants)
}

func (h *Handler) CreateMerchant(c *gin.Context) {
	var req struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		ContactName  string `json:"contact_name"`
		ContactPhone string `json:"contact_phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "name, email, password are required")
		return
	}

	m := model.Merchant{
		Name:         req.Name,
		Email:        req.Email,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		Status:       "active",
	}
	if err := m.SetPassword(req.Password); err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "failed to hash password")
		return
	}

	if err := h.db.Create(&m).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	// Don't return password hash
	m.PasswordHash = ""
	response.Success(c, m)
}

func (h *Handler) GetMerchant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid merchant id")
		return
	}
	var m model.Merchant
	if err := h.db.First(&m, "id = ?", id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "merchant not found")
		return
	}
	m.PasswordHash = ""
	response.Success(c, m)
}

func (h *Handler) UpdateMerchant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid merchant id")
		return
	}

	var req struct {
		Name         *string `json:"name"`
		Email        *string `json:"email"`
		Password     *string `json:"password"`
		ContactName  *string `json:"contact_name"`
		ContactPhone *string `json:"contact_phone"`
		Status       *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.ContactName != nil {
		updates["contact_name"] = *req.ContactName
	}
	if req.ContactPhone != nil {
		updates["contact_phone"] = *req.ContactPhone
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Password != nil {
		var m model.Merchant
		if err := h.db.First(&m, "id = ?", id).Error; err != nil {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "merchant not found")
			return
		}
		if err := m.SetPassword(*req.Password); err != nil {
			response.Error(c, http.StatusInternalServerError, "INTERNAL", "failed to hash password")
			return
		}
		updates["password_hash"] = m.PasswordHash
	}

	if err := h.db.Model(&model.Merchant{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	response.Success(c, gin.H{"updated": true})
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

	var wechatCbs []model.WechatPayCallback
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

	// Convert yuan (string) to cents (int64) for the service layer
	amountYuan := 0.0
	if _, err := fmt.Sscanf(req.RefundAmount, "%f", &amountYuan); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid refund_amount")
		return
	}
	amountCents := int64(amountYuan * 100)

	result, err := h.payService.Refund(c.Request.Context(), &service.RefundInput{
		TradeNo:      req.TradeNo,
		RefundAmount: amountCents,
		RefundReason: req.RefundReason,
	})
	if err != nil {
		response.Error(c, http.StatusBadGateway, "REFUND_FAILED", err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":         "refund success",
		"channel":         result.Payment.Channel,
		"refund_id":       result.Refund.ID.String(),
		"refund_fee":      result.Refund.RefundFee,
		"channel_refund_id": result.Refund.ChannelRefundID,
	})
}

// ---- Subscription Plans ----

func (h *Handler) ListPlans(c *gin.Context) {
	plans, err := h.planRepo.ListAll()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	response.Success(c, plans)
}

func (h *Handler) CreatePlan(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		MerchantID  string `json:"merchant_id"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
		Interval    string `json:"interval"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.Name == "" || req.Amount <= 0 || req.Interval == "" || req.MerchantID == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "name, amount, interval, merchant_id are required")
		return
	}
	if req.Currency == "" {
		req.Currency = "CNY"
	}

	merchantID, err := uuid.Parse(req.MerchantID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid merchant_id")
		return
	}

	plan := &model.SubscriptionPlan{
		MerchantID:  merchantID,
		Name:        req.Name,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Interval:    req.Interval,
		Description: req.Description,
		Status:      model.PlanStatusActive,
	}
	if err := h.planRepo.Create(plan); err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	response.Success(c, plan)
}

func (h *Handler) UpdatePlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid plan id")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Amount      *int64  `json:"amount"`
		Currency    *string `json:"currency"`
		Interval    *string `json:"interval"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Amount != nil {
		updates["amount"] = *req.Amount
	}
	if req.Currency != nil {
		updates["currency"] = *req.Currency
	}
	if req.Interval != nil {
		updates["interval"] = *req.Interval
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "no fields to update")
		return
	}

	if err := h.planRepo.Update(id, updates); err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	response.Success(c, gin.H{"updated": true})
}
