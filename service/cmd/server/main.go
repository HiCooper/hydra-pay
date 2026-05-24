package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hydra/pay-service/internal/channel"
	"github.com/hydra/pay-service/internal/channel/alipay"
	"github.com/hydra/pay-service/internal/channel/wechat"
	"github.com/hydra/pay-service/internal/config"
	"github.com/hydra/pay-service/internal/database"
	"github.com/hydra/pay-service/internal/model"
	"github.com/hydra/pay-service/internal/router"
	"github.com/hydra/pay-service/internal/service"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	r := router.Setup(cfg, db)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	go func() {
		log.Printf("[hydra-pay] starting on :%s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	go runOrderSync(db, cfg)
	go runTaskCleanup(db)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[hydra-pay] shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("[hydra-pay] server exited")
}

func runOrderSync(db *gorm.DB, cfg *config.Config) {
	getAdapter := func(ch string) (channel.Adapter, error) {
		switch ch {
		case model.ChannelAlipay:
			return alipay.NewAdapter(&cfg.Alipay)
		case model.ChannelWechat:
			return wechat.NewAdapter(&cfg.Wechat)
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