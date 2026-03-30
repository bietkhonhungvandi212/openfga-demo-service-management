package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	ratePerSec float64
	lastRefill time.Time
}

type RateLimiter struct {
	buckets    map[string]*TokenBucket
	mu         sync.RWMutex
	ratePerSec float64
	burst      int
	enabled    bool
}

func NewRateLimiter(ratePerSec float64, burst int, enabled bool) *RateLimiter {
	return &RateLimiter{
		buckets:    make(map[string]*TokenBucket),
		ratePerSec: ratePerSec,
		burst:      burst,
		enabled:    enabled,
	}
}

func (rl *RateLimiter) Allow(key string) (bool, int, int64) {
	if !rl.enabled {
		return true, rl.burst, 0
	}

	rl.mu.Lock()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &TokenBucket{
			tokens:     float64(rl.burst),
			maxTokens:  float64(rl.burst),
			ratePerSec: rl.ratePerSec,
			lastRefill: time.Now(),
		}
		rl.buckets[key] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * bucket.ratePerSec
	if bucket.tokens > bucket.maxTokens {
		bucket.tokens = bucket.maxTokens
	}
	bucket.lastRefill = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, int(bucket.tokens), now.Unix()
	}

	return false, 0, now.Add(time.Duration(1/bucket.ratePerSec) * time.Second).Unix()
}

func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.enabled {
			c.Next()
			return
		}

		key := c.ClientIP()
		allowed, remaining, reset := limiter.Allow(key)

		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.burst))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(reset, 10))

		if !allowed {
			RecordRateLimitExceeded()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
