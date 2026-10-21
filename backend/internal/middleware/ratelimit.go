package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// simpleTokenBucket is an in-memory rate limiter suitable for single-instance deployments.
// For multi-instance production, replace with Redis-backed limits.
type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	rate      float64 // tokens per second
	capacity  float64
	lastSweep time.Time
}

// sweepEvery is how often buckets that have refilled completely are dropped, so the map only
// holds recently active clients.
const sweepEvery = time.Minute

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	if requestsPerMinute < 1 {
		requestsPerMinute = 60
	}
	r := float64(requestsPerMinute) / 60.0
	return &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     r,
		capacity: float64(requestsPerMinute),
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	if now.Sub(rl.lastSweep) >= sweepEvery {
		refill := time.Duration(rl.capacity / rl.rate * float64(time.Second))
		for k, b := range rl.buckets {
			if now.Sub(b.lastRefill) >= refill {
				delete(rl.buckets, k) // full again: same as a new bucket
			}
		}
		rl.lastSweep = now
	}
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &bucket{tokens: rl.capacity - 1, lastRefill: now}
		return true
	}
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}
	b.lastRefill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RateLimitByIP applies a per-IP limit.
func RateLimitByIP(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !rl.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, try again later",
			})
			return
		}
		c.Next()
	}
}

// RateLimitLogin tighter limit for auth endpoints (key = IP + identifier hash prefix).
func RateLimitLogin(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "login:" + c.ClientIP()
		if !rl.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many login attempts, try again later",
			})
			return
		}
		c.Next()
	}
}
