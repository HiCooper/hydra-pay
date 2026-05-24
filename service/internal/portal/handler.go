package portal

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/pkg/response"
)

// Use the same context key as middleware.APIKeyAuth
const contextAppID = "app_id"

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

// getAppID extracts the authenticated app ID from the Gin context.
func getAppID(c *gin.Context) uuid.UUID {
	id, _ := c.Get(contextAppID)
	return id.(uuid.UUID)
}

// Me returns the current app's information.
func (h *Handler) Me(c *gin.Context) {
	var app model.App
	if err := h.db.First(&app, "id = ?", getAppID(c)).Error; err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "app not found")
		return
	}
	// Don't expose full API key in response
	response.Success(c, gin.H{
		"id":              app.ID,
		"name":            app.Name,
		"api_key_preview": app.APIKey[:8] + "..." + app.APIKey[len(app.APIKey)-4:],
		"api_key_full":    app.APIKey,
		"status":          app.Status,
		"alipay_pid":      app.AlipayPID,
		"wechat_sub_mchid": app.WechatSubMchid,
		"wechat_sub_appid": app.WechatSubAppid,
		"webhook_url":     app.WebhookURL,
		"created_at":      app.CreatedAt,
	})
}

// Dashboard returns app-specific stats.
func (h *Handler) Dashboard(c *gin.Context) {
	appID := getAppID(c)
	today := time.Now().Truncate(24 * time.Hour)

	var todayOrders, todayPaid int64
	h.db.Model(&model.Payment{}).Where("app_id = ? AND created_at >= ?", appID, today).Count(&todayOrders)
	h.db.Model(&model.Payment{}).Where("app_id = ? AND created_at >= ? AND status = ?", appID, today, model.PaymentStatusPaid).Count(&todayPaid)

	var todayRevenue float64
	h.db.Model(&model.Payment{}).
		Where("app_id = ? AND created_at >= ? AND status = ?", appID, today, model.PaymentStatusPaid).
		Select("COALESCE(SUM(amount), 0)").Row().Scan(&todayRevenue)

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

// Orders returns this app's payment orders.
func (h *Handler) Orders(c *gin.Context) {
	appID := getAppID(c)
	var payments []model.Payment
	var total int64

	query := h.db.Model(&model.Payment{}).Where("app_id = ?", appID)
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

// OrderDetail returns a single order with its events.
func (h *Handler) OrderDetail(c *gin.Context) {
	appID := getAppID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid payment id")
		return
	}

	payment, err := h.paymentRepo.GetByID(id)
	if err != nil || payment.AppID != appID {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "payment not found")
		return
	}

	events, _ := h.eventRepo.ListByPayment(id)

	var alipayCbs []model.AlipayCallback
	h.db.Where("payment_id = ?", id).Order("created_at DESC").Find(&alipayCbs)

	var wechatCbs []model.WeChatCallback
	h.db.Where("payment_id = ?", id).Order("created_at DESC").Find(&wechatCbs)

	response.Success(c, gin.H{
		"payment":         payment,
		"events":          events,
		"alipay_callbacks": alipayCbs,
		"wechat_callbacks": wechatCbs,
	})
}

// Events returns this app's payment events.
func (h *Handler) Events(c *gin.Context) {
	appID := getAppID(c)
	var events []model.PaymentEvent

	// Join with payments to scope by app_id
	h.db.Model(&model.PaymentEvent{}).
		Joins("JOIN payments ON payments.id = payment_events.payment_id").
		Where("payments.app_id = ?", appID).
		Order("payment_events.created_at DESC").
		Limit(50).
		Find(&events)

	response.Success(c, events)
}

// UpdateSettings updates the app's settings.
func (h *Handler) UpdateSettings(c *gin.Context) {
	appID := getAppID(c)
	var req struct {
		WebhookURL     *string `json:"webhook_url"`
		AlipayPID      *string `json:"alipay_pid"`
		WechatSubMchid *string `json:"wechat_sub_mchid"`
		WechatSubAppid *string `json:"wechat_sub_appid"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.WebhookURL != nil {
		updates["webhook_url"] = *req.WebhookURL
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

	if err := h.db.Model(&model.App{}).Where("id = ?", appID).Updates(updates).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	var app model.App
	h.db.First(&app, "id = ?", appID)
	response.Success(c, app)
}
