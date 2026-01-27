package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs all incoming requests for audit trail
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get client info
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		
		// Get user from context (if authenticated)
		userEmail := "anonymous"
		if email, exists := c.Get("user_email"); exists {
			userEmail = email.(string)
		}

		// Build query string
		if raw != "" {
			path = path + "?" + raw
		}

		// Log format: [timestamp] user | IP | method | path | status | latency
		log.Printf("[AUDIT] %s | %s | %s | %s | %d | %v",
			userEmail,
			clientIP,
			method,
			path,
			statusCode,
			latency,
		)
	}
}
