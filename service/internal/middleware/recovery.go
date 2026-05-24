package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

func StructuredRecovery() gin.HandlerFunc {
	logger := log.New(os.Stdout, "", 0)
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				entry := map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
					"level":     "fatal",
					"error":     r,
					"stack":     stack,
					"path":      c.Request.URL.Path,
					"method":    c.Request.Method,
				}
				b, _ := json.Marshal(entry)
				logger.Println(string(b))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}
