package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/middleware"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/internal/service"
	"github.com/hydra/pay-service/pkg/errors"
	"github.com/hydra/pay-service/pkg/response"
)

type SubscriptionHandler struct {
	subRepo  *repository.SubscriptionRepository
	planRepo *repository.SubscriptionPlanRepository
	db       *gorm.DB
}

func NewSubscriptionHandler(db *gorm.DB) *SubscriptionHandler {
	return &SubscriptionHandler{
		subRepo:  repository.NewSubscriptionRepository(db),
		planRepo: repository.NewSubscriptionPlanRepository(db),
		db:       db,
	}
}

// CreateSubscription creates a new subscription for a user to a plan.
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	var req struct {
		PlanID string `json:"plan_id"`
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid request body")
		return
	}
	if req.PlanID == "" || req.UserID == "" {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "plan_id and user_id are required")
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "invalid plan_id")
		return
	}

	plan, err := h.planRepo.GetByID(planID)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "subscription plan not found")
		return
	}
	if plan.Status != model.PlanStatusActive {
		response.Error(c, http.StatusBadRequest, errors.ValidationError, "subscription plan is not active")
		return
	}

	now := time.Now()
	sub := &model.Subscription{
		AppID:              appID.(uuid.UUID),
		UserID:             req.UserID,
		PlanID:             plan.ID.String(),
		Status:             model.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   service.CalculatePeriodEnd(now, plan.Interval),
	}

	if err := h.subRepo.Create(sub); err != nil {
		response.Error(c, http.StatusInternalServerError, errors.InternalError, "failed to create subscription")
		return
	}

	response.Success(c, gin.H{
		"id":                   sub.ID.String(),
		"plan_id":              sub.PlanID,
		"plan_name":            plan.Name,
		"user_id":              sub.UserID,
		"status":               sub.Status,
		"current_period_start": sub.CurrentPeriodStart,
		"current_period_end":   sub.CurrentPeriodEnd,
		"amount":               plan.Amount,
		"currency":             plan.Currency,
		"interval":             plan.Interval,
		"created_at":           sub.CreatedAt,
	})
}

// GetSubscription returns a single subscription by ID.
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid subscription id")
		return
	}

	sub, err := h.subRepo.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "subscription not found")
		return
	}
	if sub.AppID != appID.(uuid.UUID) {
		response.Error(c, http.StatusNotFound, errors.NotFound, "subscription not found")
		return
	}

	response.Success(c, sub)
}

// ListSubscriptions returns subscriptions, optionally filtered by user_id.
func (h *SubscriptionHandler) ListSubscriptions(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	userID := c.Query("user_id")
	if userID != "" {
		subs, err := h.subRepo.ListByUser(appID.(uuid.UUID), userID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, errors.InternalError, "failed to list subscriptions")
			return
		}
		response.Success(c, gin.H{"subscriptions": subs})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	subs, total, err := h.subRepo.ListByApp(appID.(uuid.UUID), page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errors.InternalError, "failed to list subscriptions")
		return
	}

	response.SuccessWithPagination(c, subs, page, pageSize, int(total))
}

// CancelSubscription cancels an active subscription.
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	appID, exists := c.Get(middleware.ContextAppID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, errors.Unauthorized, "authentication required")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "invalid subscription id")
		return
	}

	sub, err := h.subRepo.GetByID(id)
	if err != nil {
		response.Error(c, http.StatusNotFound, errors.NotFound, "subscription not found")
		return
	}
	if sub.AppID != appID.(uuid.UUID) {
		response.Error(c, http.StatusNotFound, errors.NotFound, "subscription not found")
		return
	}
	if sub.Status != model.SubscriptionStatusActive && sub.Status != model.SubscriptionStatusPastDue {
		response.Error(c, http.StatusBadRequest, "INVALID_STATE", "only active or past_due subscriptions can be cancelled")
		return
	}

	if err := h.subRepo.Cancel(id); err != nil {
		response.Error(c, http.StatusInternalServerError, errors.InternalError, "failed to cancel subscription")
		return
	}

	response.Success(c, gin.H{"status": "cancelled"})
}
