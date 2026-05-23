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
)

func Setup(cfg *config.Config, db *gorm.DB) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	paymentHandler := handler.NewPaymentHandler(db, cfg)

	v1 := r.Group("/v1")
	v1.Use(middleware.APIKeyAuth(db))
	{
		v1.POST("/payments/create", paymentHandler.CreatePayment)
		v1.GET("/payments/:id", paymentHandler.GetPayment)
	}

	callbacks := r.Group("/v1/payments/callback")
	{
		callbacks.POST("/:channel", paymentHandler.Callback)
		callbacks.POST("", paymentHandler.Callback)
	}

	// Admin API
	adminHandler := admin.NewHandler(db)
	admAPI := r.Group("/api/admin")
	admAPI.Use(admin.AdminAuth())
	{
		admAPI.GET("/dashboard", adminHandler.Dashboard)
		admAPI.GET("/config", adminHandler.ChannelConfig)

		admAPI.GET("/apps", adminHandler.ListApps)
		admAPI.POST("/apps", adminHandler.CreateApp)
		admAPI.GET("/apps/:id", adminHandler.GetApp)
		admAPI.PUT("/apps/:id", adminHandler.UpdateApp)

		admAPI.GET("/orders", adminHandler.ListOrders)
		admAPI.GET("/orders/export", adminHandler.ExportOrders)
		admAPI.GET("/orders/:id", adminHandler.GetOrder)

		admAPI.GET("/events", adminHandler.ListEvents)

		admAPI.POST("/tools/simulate-callback", adminHandler.SimulateCallback)
		admAPI.POST("/tools/test-webhook", adminHandler.TestWebhook)
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
	}

	// Developer Portal frontend SPA
	r.StaticFile("/portal", "../portal/dist/index.html")
	r.StaticFS("/portal/assets", http.Dir("../portal/dist/assets"))
	r.StaticFile("/portal/favicon.svg", "../portal/dist/favicon.svg")

	// Admin frontend SPA
	r.StaticFile("/admin", "../admin/dist/index.html")
	r.StaticFS("/admin/assets", http.Dir("../admin/dist/assets"))
	r.StaticFile("/admin/favicon.svg", "../admin/dist/favicon.svg")

	// SPA fallback for both /admin and /portal
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
		c.JSON(404, gin.H{"error": "not found"})
	})

	return r
}