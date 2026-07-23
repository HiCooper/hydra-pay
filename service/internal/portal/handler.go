package portal

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/pkg/response"
)

type Handler struct {
	db             *gorm.DB
	cfg            *config.Config
	paymentRepo    *repository.PaymentRepository
	eventRepo      *repository.EventRepository
	sessionRepo    *repository.CheckoutSessionRepository
	subRepo        *repository.SubscriptionRepository
}

func NewHandler(db *gorm.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:             db,
		cfg:            cfg,
		paymentRepo:    repository.NewPaymentRepository(db),
		eventRepo:      repository.NewEventRepository(db),
		sessionRepo:    repository.NewCheckoutSessionRepository(db),
		subRepo:        repository.NewSubscriptionRepository(db),
	}
}

func getMerchantID(c *gin.Context) uuid.UUID {
	id, _ := c.Get(merchantContextKey)
	return id.(uuid.UUID)
}

// appBelongsToMerchant checks if an app belongs to the current merchant.
func (h *Handler) appBelongsToMerchant(appID, merchantID uuid.UUID) bool {
	var count int64
	h.db.Model(&model.App{}).Where("id = ? AND merchant_id = ?", appID, merchantID).Count(&count)
	return count > 0
}

// merchantAppIDs returns all app IDs belonging to a merchant.
func (h *Handler) merchantAppIDs(merchantID uuid.UUID) []uuid.UUID {
	var apps []model.App
	h.db.Where("merchant_id = ?", merchantID).Find(&apps)
	ids := make([]uuid.UUID, len(apps))
	for i, a := range apps {
		ids[i] = a.ID
	}
	return ids
}

// ---- Me / Apps ----

func (h *Handler) Me(c *gin.Context) {
	merchantID := getMerchantID(c)
	var m model.Merchant
	if err := h.db.First(&m, "id = ?", merchantID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "merchant not found")
		return
	}
	m.PasswordHash = ""

	var apps []model.App
	h.db.Where("merchant_id = ?", merchantID).Order("created_at DESC").Find(&apps)

	response.Success(c, gin.H{
		"merchant": m,
		"apps":     apps,
	})
}

func (h *Handler) ListApps(c *gin.Context) {
	merchantID := getMerchantID(c)
	var apps []model.App
	h.db.Where("merchant_id = ?", merchantID).Order("created_at DESC").Find(&apps)
	response.Success(c, apps)
}

func (h *Handler) CreateApp(c *gin.Context) {
	merchantID := getMerchantID(c)
	var req struct {
		Name       string `json:"name"`
		WebhookURL string `json:"webhook_url"`
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
		MerchantID: merchantID,
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

// ---- Dashboard ----

func (h *Handler) Dashboard(c *gin.Context) {
	merchantID := getMerchantID(c)
	appIDs := h.merchantAppIDs(merchantID)
	today := time.Now().Truncate(24 * time.Hour)

	var todayOrders, todayPaid int64
	h.db.Model(&model.Payment{}).Where("app_id IN ? AND created_at >= ?", appIDs, today).Count(&todayOrders)
	h.db.Model(&model.Payment{}).Where("app_id IN ? AND created_at >= ? AND status = ?", appIDs, today, model.PaymentStatusPaid).Count(&todayPaid)

	var todayRevenue float64
	if len(appIDs) > 0 {
		h.db.Model(&model.Payment{}).
			Where("app_id IN ? AND created_at >= ? AND status = ?", appIDs, today, model.PaymentStatusPaid).
			Select("COALESCE(SUM(amount), 0)").Row().Scan(&todayRevenue)
	}

	successRate := float64(0)
	if todayOrders > 0 {
		successRate = float64(todayPaid) / float64(todayOrders) * 100
	}

	response.Success(c, gin.H{
		"today_orders":  todayOrders,
		"today_paid":    todayPaid,
		"today_revenue": todayRevenue / 100,
		"success_rate":  successRate,
	})
}

// ---- Orders ----

func (h *Handler) Orders(c *gin.Context) {
	appIDs := h.merchantAppIDs(getMerchantID(c))
	var payments []model.Payment
	var total int64

	query := h.db.Model(&model.Payment{}).Where("app_id IN ?", appIDs)
	if channel := c.Query("channel"); channel != "" {
		query = query.Where("channel = ?", channel)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Order("created_at DESC").Limit(50).Find(&payments)

	response.Success(c, gin.H{"orders": payments, "total": total})
}

func (h *Handler) OrderDetail(c *gin.Context) {
	merchantID := getMerchantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid payment id")
		return
	}

	payment, err := h.paymentRepo.GetByID(id)
	if err != nil || !h.appBelongsToMerchant(payment.AppID, merchantID) {
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

	response.Success(c, gin.H{
		"payment":          payment,
		"events":           events,
		"alipay_callbacks": alipayCbs,
		"wechat_callbacks": wechatCbs,
		"refunds":          refunds,
	})
}

// ---- Events ----

func (h *Handler) Events(c *gin.Context) {
	appIDs := h.merchantAppIDs(getMerchantID(c))
	var events []model.PaymentEvent
	h.db.Model(&model.PaymentEvent{}).
		Joins("JOIN payments ON payments.id = payment_events.payment_id").
		Where("payments.app_id IN ?", appIDs).
		Order("payment_events.created_at DESC").
		Limit(50).
		Find(&events)
	response.Success(c, events)
}

// ---- Settings ----

func (h *Handler) UpdateSettings(c *gin.Context) {
	merchantID := getMerchantID(c)
	var req struct {
		ContactName  *string `json:"contact_name"`
		ContactPhone *string `json:"contact_phone"`
		Password     *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.ContactName != nil {
		updates["contact_name"] = *req.ContactName
	}
	if req.ContactPhone != nil {
		updates["contact_phone"] = *req.ContactPhone
	}
	if req.Password != nil {
		var m model.Merchant
		if err := h.db.First(&m, "id = ?", merchantID).Error; err != nil {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "merchant not found")
			return
		}
		if err := m.SetPassword(*req.Password); err != nil {
			response.Error(c, http.StatusInternalServerError, "INTERNAL", "failed to hash password")
			return
		}
		updates["password_hash"] = m.PasswordHash
	}

	if len(updates) > 0 {
		if err := h.db.Model(&model.Merchant{}).Where("id = ?", merchantID).Updates(updates).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}

	var m model.Merchant
	h.db.First(&m, "id = ?", merchantID)
	m.PasswordHash = ""
	response.Success(c, m)
}

// ---- Payment Links ----

func (h *Handler) ListPaymentLinks(c *gin.Context) {
	appIDs := h.merchantAppIDs(getMerchantID(c))

	// Expire stale sessions before listing
	h.db.Model(&model.CheckoutSession{}).
		Where("app_id IN ? AND status = ? AND expires_at <= ?", appIDs, model.CheckoutSessionOpen, time.Now()).
		Update("status", model.CheckoutSessionExpired)

	var sessions []model.CheckoutSession
	h.db.Where("app_id IN ?", appIDs).Order("created_at DESC").Limit(100).Find(&sessions)

	type linkItem struct {
		ID          string `json:"id"`
		AppID       string `json:"app_id"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
		Description string `json:"description"`
		Status      string `json:"status"`
		SuccessURL  string `json:"success_url"`
		CancelURL   string `json:"cancel_url"`
		ExpiresAt   string `json:"expires_at"`
		CreatedAt   string `json:"created_at"`
		CheckoutURL string `json:"checkout_url"`
	}

	links := make([]linkItem, 0, len(sessions))
	for _, s := range sessions {
		links = append(links, linkItem{
			ID:          s.ID.String(),
			AppID:       s.AppID.String(),
			Amount:      s.Amount,
			Currency:    s.Currency,
			Description: s.Description,
			Status:      s.Status,
			SuccessURL:  s.SuccessURL,
			CancelURL:   s.CancelURL,
			ExpiresAt:   s.ExpiresAt.Format(time.RFC3339),
			CreatedAt:   s.CreatedAt.Format(time.RFC3339),
			CheckoutURL: "/pay/v2/checkout/" + s.ID.String(),
		})
	}
	response.Success(c, gin.H{"payment_links": links})
}

func (h *Handler) CreatePaymentLink(c *gin.Context) {
	merchantID := getMerchantID(c)
	var req struct {
		AppID       string `json:"app_id"`
		Amount      int64  `json:"amount"`
		Currency    string `json:"currency"`
		Description string `json:"description"`
		SuccessURL  string `json:"success_url"`
		CancelURL   string `json:"cancel_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.Amount <= 0 {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "amount must be positive")
		return
	}
	if req.AppID == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "app_id is required")
		return
	}
	appID, err := uuid.Parse(req.AppID)
	if err != nil || !h.appBelongsToMerchant(appID, merchantID) {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", "invalid app_id")
		return
	}
	if req.Currency == "" {
		req.Currency = "CNY"
	}

	session := &model.CheckoutSession{
		AppID:       appID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		SuccessURL:  req.SuccessURL,
		CancelURL:   req.CancelURL,
		Status:      model.CheckoutSessionOpen,
		ExpiresAt:   time.Now().Add(30 * time.Minute),
	}
	if err := h.sessionRepo.Create(session); err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	response.Success(c, gin.H{
		"id":           session.ID.String(),
		"amount":       session.Amount,
		"currency":     session.Currency,
		"description":  session.Description,
		"status":       session.Status,
		"success_url":  session.SuccessURL,
		"cancel_url":   session.CancelURL,
		"expires_at":   session.ExpiresAt.Format(time.RFC3339),
		"created_at":   session.CreatedAt.Format(time.RFC3339),
		"checkout_url": "/pay/v2/checkout/" + session.ID.String(),
	})
}

func (h *Handler) ExpirePaymentLink(c *gin.Context) {
	merchantID := getMerchantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return
	}
	session, err := h.sessionRepo.GetByID(id)
	if err != nil || !h.appBelongsToMerchant(session.AppID, merchantID) {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "payment link not found")
		return
	}
	if session.Status != model.CheckoutSessionOpen {
		response.Error(c, http.StatusBadRequest, "INVALID_STATE", "only open links can be expired")
		return
	}
	if err := h.sessionRepo.Expire(id); err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	response.Success(c, gin.H{"status": "expired"})
}

func (h *Handler) DeletePaymentLink(c *gin.Context) {
	merchantID := getMerchantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid id")
		return
	}
	session, err := h.sessionRepo.GetByID(id)
	if err != nil || !h.appBelongsToMerchant(session.AppID, merchantID) {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "payment link not found")
		return
	}
	if session.Status != model.CheckoutSessionExpired {
		response.Error(c, http.StatusBadRequest, "INVALID_STATE", "only expired links can be deleted")
		return
	}
	if err := h.sessionRepo.Delete(id); err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ---- Subscriptions ----

func (h *Handler) ListSubscriptions(c *gin.Context) {
	appIDs := h.merchantAppIDs(getMerchantID(c))
	var subs []model.Subscription
	if len(appIDs) > 0 {
		h.db.Where("app_id IN ?", appIDs).Order("created_at DESC").Limit(100).Find(&subs)
	}
	if subs == nil {
		subs = []model.Subscription{}
	}
	response.Success(c, gin.H{"subscriptions": subs})
}

	func (h *Handler) ListChannels(c *gin.Context) {
		type ChannelInfo struct {
			Key        string `json:"key"`
			Label      string `json:"label"`
			Configured bool   `json:"configured"`
		}

		var pcs []model.PaymentChannel
		if err := h.db.Where("enabled = true").Order("sort_order ASC").Find(&pcs).Error; err != nil {
			response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}

		configured := func(channel string) bool {
			switch channel {
			case model.ChannelAlipay:
				return h.cfg.Alipay.AppID != "" && h.cfg.Alipay.PrivateKey != ""
			case model.ChannelWechat:
				return h.cfg.Wechat.MchID != "" && h.cfg.Wechat.APIv3Key != ""
			case model.ChannelUnionpay:
				return h.cfg.Unionpay.AppID != "" && h.cfg.Unionpay.Secret != ""
			case model.ChannelEcny:
				return h.cfg.Ecny.AppID != "" && h.cfg.Ecny.PrivateKey != ""
			}
			return false
		}

		channels := make([]ChannelInfo, 0, len(pcs))
		for _, pc := range pcs {
			channels = append(channels, ChannelInfo{
				Key:        pc.Key,
				Label:      pc.Label,
				Configured: configured(pc.Key),
			})
		}

		response.Success(c, gin.H{"channels": channels})
	}
