package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/channel/alipay"
	"github.com/hydra/pay-service/internal/channel/ecny"
	"github.com/hydra/pay-service/internal/channel/unionpay"
	"github.com/hydra/pay-service/internal/channel/wechat"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/database"
	"github.com/hydra/pay-service/internal/middleware"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/router"
	"github.com/hydra/pay-service/pkg/logger"
	"github.com/hydra/pay-service/pkg/metrics"
	"github.com/hydra/pay-service/pkg/telemetry"
	"github.com/hydra/pay-service/internal/service"
	"gorm.io/gorm"
)

var ready atomic.Bool

func main() {
	ready.Store(true)

	logger.Init()
	metrics.Init()

	if err := telemetry.Init(); err != nil {
		log.Printf("WARNING: telemetry init failed (tracing disabled): %v", err)
	}

	cfg := config.Load()
	_ = cfg.Validate() // logs warnings, non-fatal

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	middleware.InitSentinel()

	r := router.Setup(cfg, db)

	// Readiness probe for graceful drain
	r.GET("/readyz", func(c *gin.Context) {
		if ready.Load() {
			c.String(200, "ok")
		} else {
			c.String(503, "shutting down")
		}
	})

	readTimeout, _ := time.ParseDuration(cfg.Server.ReadTimeout)
	writeTimeout, _ := time.ParseDuration(cfg.Server.WriteTimeout)
	idleTimeout, _ := time.ParseDuration(cfg.Server.IdleTimeout)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	go func() {
		logger.Info(context.Background(), "server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	go runOrderSync(db, cfg)
	go runTaskCleanup(db)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(context.Background(), "shutting down...")

	// Signal readiness failure so load balancers drain traffic
	ready.Store(false)

	// Allow LB health checks to propagate (drain period)
	drainWait := 5 * time.Second
	logger.Info(context.Background(), "draining connections", "wait", drainWait)
	time.Sleep(drainWait)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close database connection pool
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
		logger.Info(context.Background(), "database connection pool closed")
	}

	if err := telemetry.Shutdown(ctx); err != nil {
		log.Printf("WARNING: telemetry shutdown failed: %v", err)
	}

	logger.Info(context.Background(), "server exited")
}

func runOrderSync(db *gorm.DB, cfg *config.Config) {
	getAdapter := func(ch string) (channel.Adapter, error) {
		switch ch {
		case model.ChannelAlipay:
			return alipay.NewAdapter(&cfg.Alipay)
		case model.ChannelWechat:
			return wechat.NewAdapter(&cfg.Wechat)
		case model.ChannelUnionpay:
			return unionpay.NewAdapter(&cfg.Unionpay)
		case model.ChannelEcny:
			return ecny.NewAdapter(&cfg.Ecny)
		default:
			return nil, fmt.Errorf("unsupported channel: %s", ch)
		}
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		service.SyncExpiredOrders(db, getAdapter)
	}
}

func runTaskCleanup(db *gorm.DB) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		db.Exec("DELETE FROM scheduled_tasks WHERE status IN ('done','cancelled') AND created_at < NOW() - INTERVAL '7 days'")
	}
}
