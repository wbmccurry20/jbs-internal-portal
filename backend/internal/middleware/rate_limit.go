package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, times := range rl.requests {
		// Remove timestamps outside the window
		valid := []time.Time{}
		for _, t := range times {
			if now.Sub(t) < rl.window {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = valid
		}
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create request list for this IP
	times, exists := rl.requests[ip]
	if !exists {
		times = []time.Time{}
	}

	// Remove old timestamps
	valid := []time.Time{}
	for _, t := range times {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	// Check if under limit
	if len(valid) >= rl.limit {
		rl.requests[ip] = valid
		return false
	}

	// Add current request
	valid = append(valid, now)
	rl.requests[ip] = valid
	return true
}

// Global rate limiters
var (
	loginLimiter      = newRateLimiter(5, time.Minute)   // 5 login attempts per minute
	uploadLimiter     = newRateLimiter(10, time.Minute)  // 10 uploads per minute
	generalLimiter    = newRateLimiter(100, time.Minute) // 100 general requests per minute
	submissionLimiter = newRateLimiter(10, time.Hour)    // 10 submissions per hour per IP
)

// RateLimitLogin restricts login attempts
func RateLimitLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !loginLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many login attempts. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitUpload restricts file uploads
func RateLimitUpload() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !uploadLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many upload requests. Please wait before uploading again.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitGeneral restricts general API calls
func RateLimitGeneral() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !generalLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please slow down.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitSubmission restricts public payment application submissions.
// Stricter than the general limiter: 10 submissions per hour per IP to deter form spam.
func RateLimitSubmission() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !submissionLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many submissions. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
