package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// QueryLogger logs slow queries for performance monitoring
func QueryLogger(slowThreshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		
		c.Next()
		
		duration := time.Since(start)
		
		// Log slow requests
		if duration > slowThreshold {
			log.Printf("⚠️  SLOW REQUEST: %s %s took %v", 
				c.Request.Method, 
				path, 
				duration,
			)
		}
		
		// Add timing header
		c.Header("X-Response-Time", duration.String())
	}
}
