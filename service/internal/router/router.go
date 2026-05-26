package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/admin"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/handler"
	"github.com/hydra/pay-service/internal/middleware"
	"github.com/hydra/pay-service/internal/portal"
	"github.com/hydra/pay-service/internal/repository"
	"github.com/hydra/pay-service/internal/service"
)

func Setup(cfg *config.Config, db *gorm.DB) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()

	r.Use(middleware.StructuredRecovery())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.CORS(cfg.Server.CORSOrigins))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	paymentHandler := handler.NewPaymentHandler(db, cfg)

	v1 := r.Group("/v1")
	v1.Use(middleware.APIKeyAuth(db))
	v1.Use(middleware.Idempotency(db))
	{
		v1.POST("/payments/create", paymentHandler.CreatePayment)
		v1.GET("/payments/:id", paymentHandler.GetPayment)

		// Checkout Session (Stripe-style)
		sessionHandler := handler.NewCheckoutSessionHandler(db)
		v1.POST("/checkout/sessions", sessionHandler.CreateSession)

		// Refunds
		refundHandler := handler.NewRefundHandler(db, cfg)
		v1.POST("/refunds", refundHandler.CreateRefund)
		v1.GET("/refunds/:id", refundHandler.GetRefund)
		v1.GET("/payments/:id/refunds", refundHandler.ListPaymentRefunds)

			// Subscriptions
			subscriptionHandler := handler.NewSubscriptionHandler(db)
			v1.POST("/subscriptions", subscriptionHandler.CreateSubscription)
			v1.GET("/subscriptions", subscriptionHandler.ListSubscriptions)
			v1.GET("/subscriptions/:id", subscriptionHandler.GetSubscription)
			v1.POST("/subscriptions/:id/cancel", subscriptionHandler.CancelSubscription)
		}

		callbacks := r.Group("/v1/payments/callback")
	{
		callbacks.POST("/:channel", paymentHandler.Callback)
		callbacks.POST("", paymentHandler.Callback)
	}

	// Onboarding callbacks — public, no auth
	onboardingCbHandler := handler.NewOnboardingCallbackHandler(db, cfg)
	onboardingCb := r.Group("/v1/onboarding/callback")
	{
		onboardingCb.POST("/:channel", onboardingCbHandler.Callback)
	}

	// Checkout API — public, no auth (for hosted checkout page)
	checkoutHandler := handler.NewCheckoutHandler(db, cfg)
	checkout := r.Group("/api/checkout")
	{
		checkout.GET("/:session_id", checkoutHandler.GetCheckout)
		checkout.GET("/:session_id/payment-status", checkoutHandler.GetPaymentStatus)
		checkout.POST("/:session_id/activate", checkoutHandler.ActivatePayment)
	}

	// Admin API
	payRepo := repository.NewPaymentRepository(db)
	payService := service.NewPaymentService(payRepo, cfg, db)
	adminHandler := admin.NewHandler(db, payService)
	admAPI := r.Group(cfg.Server.AdminAPIPath)
	admAPI.Use(admin.AdminAuth())
	{
		admAPI.GET("/dashboard", adminHandler.Dashboard)
		admAPI.GET("/config", adminHandler.ChannelConfig)

		admAPI.GET("/apps", adminHandler.ListApps)
		admAPI.POST("/apps", adminHandler.CreateApp)
		admAPI.GET("/apps/:id", adminHandler.GetApp)
		admAPI.PUT("/apps/:id", adminHandler.UpdateApp)
		admAPI.POST("/apps/:id/onboard", adminHandler.InitiateOnboarding)
		admAPI.GET("/apps/:id/onboarding", adminHandler.GetOnboardingStatus)

		admAPI.GET("/onboarding", adminHandler.ListOnboardings)

		admAPI.GET("/orders", adminHandler.ListOrders)
		admAPI.GET("/orders/export", adminHandler.ExportOrders)
		admAPI.GET("/orders/:id", adminHandler.GetOrder)

			admAPI.GET("/plans", adminHandler.ListPlans)
			admAPI.POST("/plans", adminHandler.CreatePlan)
			admAPI.PUT("/plans/:id", adminHandler.UpdatePlan)

			admAPI.GET("/events", adminHandler.ListEvents)

		admAPI.POST("/tools/simulate-callback", adminHandler.SimulateCallback)
		admAPI.POST("/tools/test-webhook", adminHandler.TestWebhook)
		admAPI.POST("/tools/test-refund", adminHandler.TestRefund)
		admAPI.GET("/tools/connectivity", adminHandler.ConnectivityCheck)
	}

	// Developer Portal API (API Key auth, scoped to app)
	portalHandler := portal.NewHandler(db)
	portalAPI := r.Group("/portal/api")
	portalAPI.Use(middleware.APIKeyAuth(db))
	{
		portalAPI.GET("/me", portalHandler.Me)
		portalAPI.GET("/dashboard", portalHandler.Dashboard)
		portalAPI.GET("/orders", portalHandler.Orders)
		portalAPI.GET("/orders/:id", portalHandler.OrderDetail)
		portalAPI.GET("/events", portalHandler.Events)
		portalAPI.PUT("/settings", portalHandler.UpdateSettings)
			portalAPI.GET("/payment-links", portalHandler.ListPaymentLinks)
			portalAPI.POST("/payment-links", portalHandler.CreatePaymentLink)
			portalAPI.POST("/payment-links/:id/expire", portalHandler.ExpirePaymentLink)
			portalAPI.DELETE("/payment-links/:id", portalHandler.DeletePaymentLink)
			portalAPI.GET("/subscriptions", portalHandler.ListSubscriptions)
		}

	// Developer Portal frontend SPA
	r.StaticFile("/portal", "../portal/dist/index.html")
	r.StaticFS("/portal/assets", http.Dir("../portal/dist/assets"))
	r.StaticFile("/portal/favicon.svg", "../portal/dist/favicon.svg")

	// Admin frontend SPA
	r.StaticFile("/admin", "../admin/dist/index.html")
	r.StaticFS("/admin/assets", http.Dir("../admin/dist/assets"))
	r.StaticFile("/admin/favicon.svg", "../admin/dist/favicon.svg")

	// Pay frontend SPA (hosted checkout page)
	r.StaticFile("/pay", "../pay-frontend/dist/index.html")
	r.StaticFS("/pay/assets", http.Dir("../pay-frontend/dist/assets"))
	r.StaticFile("/pay/favicon.svg", "../pay-frontend/dist/favicon.svg")
	r.StaticFile("/pay/alipay_logo.svg", "../pay-frontend/dist/alipay_logo.svg")
	r.StaticFile("/pay/wechat_pay_logo.svg", "../pay-frontend/dist/wechat_pay_logo.svg")

		// SDK and embed assets
		r.StaticFS("/sdk", http.Dir("../pay-frontend/public"))

		// SPA fallback for /admin, /portal, and /pay
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/admin") {
			c.File("../admin/dist/index.html")
			return
		}
		if strings.HasPrefix(path, "/portal") {
			c.File("../portal/dist/index.html")
			return
		}
		if strings.HasPrefix(path, "/pay") {
			c.File("../pay-frontend/dist/index.html")
			return
		}
		c.JSON(404, gin.H{"error": "not found"})
	})

	return r
}