package middleware

import (
	"context"
	"crypto/md5"
	"daily-notes-api/pkg/cache"
	"daily-notes-api/pkg/response"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	})
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				c.JSON(500, gin.H{
					"success": false,
					"message": "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// CacheMiddleware creates a middleware for caching GET requests
func CacheMiddleware(cacheService *cache.CacheService, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only cache GET requests
		if c.Request.Method != "GET" || cacheService == nil {
			c.Next()
			return
		}

		// Generate cache key based on path, query params, and user ID
		cacheKey := generateCacheKey(c)
		if cacheKey == "" {
			c.Next()
			return
		}

		// Try to get from cache
		ctx, cancel := context.WithTimeout(c, 5*time.Second)
		defer cancel()

		var cachedResponse response.Response
		err := cacheService.Get(ctx, cacheKey, &cachedResponse)
		if err == nil {
			log.Printf("Cache hit: %s", cacheKey)
			c.JSON(200, cachedResponse)
			c.Abort()
			return
		}

		// Create a custom response writer to capture the response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
		}
		c.Writer = writer

		// Process the request
		c.Next()

		// Cache successful responses
		if c.Writer.Status() == 200 && len(writer.body) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Parse the response to cache it
			var resp response.Response
			if err := json.Unmarshal(writer.body, &resp); err == nil {
				if err := cacheService.Set(ctx, cacheKey, resp, ttl); err != nil {
					log.Printf("Failed to cache response: %v", err)
				} else {
					log.Printf("Cached response: %s", cacheKey)
				}
			}
		}
	}
}

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body []byte
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return w.ResponseWriter.Write(data)
}

// generateCacheKey creates a unique cache key based on request path, params, and user
func generateCacheKey(c *gin.Context) string {
	// Get user ID from context
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return "" // Don't cache non-authenticated requests
	}

	// Build base key with path and user ID
	path := c.Request.URL.Path
	keyParts := []string{
		"api_cache",
		path,
		fmt.Sprintf("user:%v", userID),
	}

	// Add sorted query parameters for consistent keys
	if len(c.Request.URL.RawQuery) > 0 {
		params, _ := url.ParseQuery(c.Request.URL.RawQuery)
		var sortedParams []string

		for key, values := range params {
			sort.Strings(values)
			for _, value := range values {
				sortedParams = append(sortedParams, fmt.Sprintf("%s:%s", key, value))
			}
		}

		sort.Strings(sortedParams)
		if len(sortedParams) > 0 {
			keyParts = append(keyParts, strings.Join(sortedParams, ","))
		}
	}

	// Join and hash for consistent length
	keyString := strings.Join(keyParts, ":")
	hash := md5.Sum([]byte(keyString))
	return fmt.Sprintf("cache:%x", hash)
}

// InvalidateCachePattern provides a helper to invalidate cache patterns
func InvalidateCachePattern(cacheService *cache.CacheService, pattern string) {
	if cacheService == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := cacheService.DeletePattern(ctx, pattern); err != nil {
		log.Printf("Failed to invalidate cache pattern %s: %v", pattern, err)
	} else {
		log.Printf("Invalidated cache pattern: %s", pattern)
	}
}
