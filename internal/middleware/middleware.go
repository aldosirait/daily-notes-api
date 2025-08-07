package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
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

// RateLimiter represents a rate limiter for specific operations
type RateLimiter struct {
	visitors map[string]*Visitor
	mutex    sync.RWMutex
	rate     int           // requests per window
	window   time.Duration // time window
	cleanup  time.Duration // cleanup interval
}

// Visitor represents a visitor with their request history
type Visitor struct {
	requests []time.Time
	lastSeen time.Time
	mutex    sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed per window
// window: time window duration
// cleanup: cleanup interval for old visitors
func NewRateLimiter(rate int, window, cleanup time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
		cleanup:  cleanup,
	}

	// Start cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// cleanupVisitors removes old visitors to prevent memory leaks
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		cutoff := time.Now().Add(-rl.cleanup)
		for ip, visitor := range rl.visitors {
			visitor.mutex.RLock()
			lastSeen := visitor.lastSeen
			visitor.mutex.RUnlock()

			if lastSeen.Before(cutoff) {
				delete(rl.visitors, ip)
			}
		}
		rl.mutex.Unlock()
	}
}

// isAllowed checks if a request from the given IP is allowed
func (rl *RateLimiter) isAllowed(ip string) (bool, int, time.Duration) {
	rl.mutex.RLock()
	visitor, exists := rl.visitors[ip]
	rl.mutex.RUnlock()

	if !exists {
		visitor = &Visitor{
			requests: make([]time.Time, 0),
			lastSeen: time.Now(),
		}
		rl.mutex.Lock()
		rl.visitors[ip] = visitor
		rl.mutex.Unlock()
	}

	visitor.mutex.Lock()
	defer visitor.mutex.Unlock()

	now := time.Now()
	visitor.lastSeen = now

	// Remove requests outside the window
	cutoff := now.Add(-rl.window)
	validRequests := visitor.requests[:0]
	for _, reqTime := range visitor.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	visitor.requests = validRequests

	// Check if limit is exceeded
	if len(visitor.requests) >= rl.rate {
		// Calculate reset time (when the oldest request expires)
		oldestRequest := visitor.requests[0]
		resetTime := oldestRequest.Add(rl.window)
		resetDuration := time.Until(resetTime)

		return false, rl.rate - len(visitor.requests), resetDuration
	}

	// Add current request
	visitor.requests = append(visitor.requests, now)
	remaining := rl.rate - len(visitor.requests)

	return true, remaining, 0
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getClientIP(c)
		allowed, remaining, resetDuration := rateLimiter.isAllowed(ip)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(rateLimiter.rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if resetDuration > 0 {
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(resetDuration).Unix(), 10))
			c.Header("Retry-After", strconv.Itoa(int(resetDuration.Seconds())))
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success":     false,
				"message":     fmt.Sprintf("Rate limit exceeded. Try again in %v", resetDuration.Round(time.Second)),
				"retry_after": int(resetDuration.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := len(xff); idx > 0 {
			if commaIdx := 0; commaIdx < idx {
				for i, char := range xff {
					if char == ',' {
						commaIdx = i
						break
					}
				}
				if commaIdx > 0 {
					return xff[:commaIdx]
				}
			}
			return xff
		}
	}

	// Check X-Real-IP header (for nginx proxy)
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to remote address
	return c.ClientIP()
}

// Global rate limiters for different endpoints
var (
	AuthRateLimiter *RateLimiter
	once            sync.Once
)

// InitRateLimiters initializes the rate limiters
func InitRateLimiters() {
	once.Do(func() {
		// Allow 5 login/register attempts per 15 minutes per IP
		AuthRateLimiter = NewRateLimiter(5, 15*time.Minute, 30*time.Minute)
	})
}

// AuthRateLimit returns the rate limiting middleware for auth endpoints
func AuthRateLimit() gin.HandlerFunc {
	InitRateLimiters()
	return RateLimitMiddleware(AuthRateLimiter)
}
