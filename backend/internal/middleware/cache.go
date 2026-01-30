package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type cacheEntry struct {
	data      []byte
	timestamp time.Time
}

var (
	cache      = make(map[string]cacheEntry)
	cacheMutex sync.RWMutex
	cacheTTL   = 5 * time.Minute
)

// CacheMiddleware caches GET requests for specified duration
func CacheMiddleware(ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only cache GET requests
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// Create cache key from path and query
		cacheKey := c.Request.URL.String()

		// Check cache
		cacheMutex.RLock()
		entry, exists := cache[cacheKey]
		cacheMutex.RUnlock()

		if exists && time.Since(entry.timestamp) < ttl {
			// Cache hit
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, "application/json", entry.data)
			c.Abort()
			return
		}

		// Cache miss - capture response
		c.Header("X-Cache", "MISS")
		
		// Create custom writer to capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:          &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		// Only cache successful responses
		if c.Writer.Status() == http.StatusOK && writer.body.Len() > 0 {
			cacheMutex.Lock()
			cache[cacheKey] = cacheEntry{
				data:      writer.body.Bytes(),
				timestamp: time.Now(),
			}
			cacheMutex.Unlock()
		}
	}
}

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// ClearCache clears all cached entries (call after POST/PUT/DELETE)
func ClearCache() {
	cacheMutex.Lock()
	cache = make(map[string]cacheEntry)
	cacheMutex.Unlock()
}

// ClearCachePattern clears cache entries matching a pattern
func ClearCachePattern(pattern string) {
	cacheMutex.Lock()
	for key := range cache {
		if len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			delete(cache, key)
		}
	}
	cacheMutex.Unlock()
}
