package middleware

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/alibaba/sentinel-golang/core/flow"
	sentinelPlugin "github.com/sentinel-group/sentinel-go-adapters/gin"

	"github.com/hydra/pay-service/pkg/logger"
)

// RateLimit returns a Sentinel-based Gin middleware that enforces per-app QPS limiting.
// Each app (identified by ContextAppID) gets its own token bucket with the given qps.
// Rules are lazily loaded on first access.
func RateLimit(qps float64) gin.HandlerFunc {
	var loaded sync.Map

	return sentinelPlugin.SentinelMiddleware(
		sentinelPlugin.WithResourceExtractor(func(c *gin.Context) string {
			appID, exists := c.Get(ContextAppID)
			if !exists {
				return "unknown"
			}
			name := "app:" + appID.(uuid.UUID).String()
			if _, ok := loaded.Load(name); !ok {
				flow.LoadRules([]*flow.Rule{
					{
						Resource:               name,
						TokenCalculateStrategy: flow.Direct,
						ControlBehavior:        flow.Reject,
						Threshold:              qps,
						StatIntervalInMs:       1000,
					},
				})
				loaded.Store(name, true)
			}
			return name
		}),
		sentinelPlugin.WithBlockFallback(func(c *gin.Context) {
			c.AbortWithStatusJSON(429, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOO_MANY_REQUESTS",
					"message": "请求过于频繁，请稍后再试",
				},
			})
		}),
	)
}

// InitSentinel initializes the Sentinel runtime with flow control and circuit breaker rules.
// Call once at startup before registering middleware.
func InitSentinel() {
	sentinel.InitDefault()

	// Circuit breaker rules for channel API calls
	circuitbreaker.LoadRules([]*circuitbreaker.Rule{
		{
			Resource:         "alipay",
			Strategy:         circuitbreaker.ErrorRatio,
			RetryTimeoutMs:   30000,
			MinRequestAmount: 5,
			StatIntervalMs:   10000,
			Threshold:        0.5,
		},
		{
			Resource:         "wechat",
			Strategy:         circuitbreaker.ErrorRatio,
			RetryTimeoutMs:   30000,
			MinRequestAmount: 5,
			StatIntervalMs:   10000,
			Threshold:        0.5,
		},
	})

	circuitbreaker.RegisterStateChangeListeners(&cbStateListener{})
}

type cbStateListener struct{}

func (l *cbStateListener) OnTransformToClosed(prev circuitbreaker.State, rule circuitbreaker.Rule) {
	logger.Info(context.Background(), "circuit breaker closed", "resource", rule.Resource, "prev", stateName(prev))
}

func (l *cbStateListener) OnTransformToOpen(prev circuitbreaker.State, rule circuitbreaker.Rule, snapshot interface{}) {
	logger.Warn(context.Background(), "circuit breaker open", "resource", rule.Resource, "prev", stateName(prev), "snapshot", snapshot)
}

func (l *cbStateListener) OnTransformToHalfOpen(prev circuitbreaker.State, rule circuitbreaker.Rule) {
	logger.Info(context.Background(), "circuit breaker half-open", "resource", rule.Resource, "prev", stateName(prev))
}

func stateName(s circuitbreaker.State) string {
	switch s {
	case circuitbreaker.Closed:
		return "closed"
	case circuitbreaker.Open:
		return "open"
	case circuitbreaker.HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
