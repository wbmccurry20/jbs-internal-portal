package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create fresh rate limiter for testing
	testLimiter := newRateLimiter(3, time.Minute)
	
	router := gin.Default()
	router.POST("/login", func(c *gin.Context) {
		ip := c.ClientIP()
		if !testLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limited"})
			c.Abort()
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("POST", "/login", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// 4th request should be rate limited
	req, _ := http.NewRequest("POST", "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.Default()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'self'")
}
