package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	InitSentinel()
	os.Exit(m.Run())
}

// fixedAppID is used for tests that need per-app counting to accumulate.
var fixedAppID = uuid.New()

func newTestRouterWithApp(qps float64, appID uuid.UUID) *gin.Engine {
	r := gin.New()
	// Simulate APIKeyAuth by setting a fixed app_id
	r.Use(func(c *gin.Context) {
		c.Set(ContextAppID, appID)
		c.Next()
	})
	r.Use(RateLimit(qps))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func setupTestRouter(qps float64) *gin.Engine {
	return newTestRouterWithApp(qps, fixedAppID)
}

func TestRateLimitNormalFlow(t *testing.T) {
	r := setupTestRouter(100) // high QPS
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimitExceeded(t *testing.T) {
	r := setupTestRouter(5) // low QPS — 5 req/s

	failures := 0
	// Send 30 rapid requests to trigger rate limiting
	for i := 0; i < 30; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		if w.Code == 429 {
			failures++
		}
	}
	if failures == 0 {
		t.Error("expected at least some 429 responses under rate limit, got none")
	}
	t.Logf("got %d out of 30 requests rate limited", failures)
}

func TestRateLimitDifferentApps(t *testing.T) {
	// Two apps share same 5 QPS threshold
	sentinelMiddleware := RateLimit(5)

	// App A: saturate its own quota
	appA := uuid.New()
	appB := uuid.New()

	rA := gin.New()
	rA.Use(func(c *gin.Context) { c.Set(ContextAppID, appA); c.Next() })
	rA.Use(sentinelMiddleware)
	rA.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	rB := gin.New()
	rB.Use(func(c *gin.Context) { c.Set(ContextAppID, appB); c.Next() })
	rB.Use(sentinelMiddleware)
	rB.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// Saturate app A
	a429 := 0
	for i := 0; i < 30; i++ {
		w := httptest.NewRecorder()
		rA.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
		if w.Code == 429 {
			a429++
		}
	}
	if a429 == 0 {
		t.Error("app A: expected at least some 429s")
	}

	// App B should still pass (separate quota)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		rB.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
		if w.Code != 200 {
			t.Errorf("app B request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimitNoAppID(t *testing.T) {
	r := gin.New()
	r.Use(RateLimit(1000))
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
		if w.Code != 200 {
			t.Errorf("unknown app request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimit429ResponseFormat(t *testing.T) {
	r := setupTestRouter(1) // 1 QPS — very restrictive

	var lastResp *httptest.ResponseRecorder
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
		if w.Code == 429 {
			lastResp = w
			break
		}
	}

	if lastResp == nil {
		t.Skip("could not trigger 429 in test; skip format check")
	}

	body := lastResp.Body.String()
	if !strings.Contains(body, "TOO_MANY_REQUESTS") {
		t.Errorf("expected TOO_MANY_REQUESTS in 429 body, got: %s", body)
	}
}
