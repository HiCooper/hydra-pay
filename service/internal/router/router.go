package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/handler"
	"github.com/hydra/pay-service/internal/middleware"
)

func Setup(cfg *config.Config, db *gorm.DB) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	paymentHandler := handler.NewPaymentHandler(db, cfg)

	// Public payment API (API Key auth)
	v1 := r.Group("/v1")
	v1.Use(middleware.APIKeyAuth(db))
	{
		v1.POST("/payments/create", paymentHandler.CreatePayment)
		v1.GET("/payments/:id", paymentHandler.GetPayment)
	}

	// Callback endpoint (no API key auth — channels use their own signature verification)
	callbacks := r.Group("/v1/payments/callback")
	{
		callbacks.POST("/:channel", paymentHandler.Callback)
		callbacks.POST("", paymentHandler.Callback) // default channel (alipay)
	}

	return r
}
