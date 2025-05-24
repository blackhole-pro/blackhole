package auth

import (
	"sync"
	"time"
)

// RateLimitConfig contains configuration for rate limiting
type RateLimitConfig struct {
	// WindowSize is the time window for rate limiting
	WindowSize time.Duration
	
	// Limits defines the maximum requests per window for different request types
	Limits map[string]int
	
	// Default limit if not specified in Limits
	DefaultLimit int
}

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		WindowSize: time.Minute,
		Limits: map[string]int{
			"challenge": 10, // 10 challenges per minute
			"verify":    20, // 20 verifications per minute
		},
		DefaultLimit: 30, // 30 requests per minute for unspecified types
	}
}

// TokenBucketRateLimiter implements a token bucket algorithm for rate limiting
type TokenBucketRateLimiter struct {
	config *RateLimitConfig
	
	// Maps key+type to bucket
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
	
	// Cleanup goroutine control
	done    chan struct{}
}

// tokenBucket represents a rate limiting bucket for a specific key and request type
type tokenBucket struct {
	tokens    int
	lastReset time.Time
	limit     int
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(config *RateLimitConfig) *TokenBucketRateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	
	limiter := &TokenBucketRateLimiter{
		config:  config,
		buckets: make(map[string]*tokenBucket),
		done:    make(chan struct{}),
	}
	
	// Start cleanup goroutine
	go limiter.cleanupLoop()
	
	return limiter
}

// AllowRequest checks if a request is allowed based on the rate limit
func (r *TokenBucketRateLimiter) AllowRequest(key string, requestType string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Create bucket key
	bucketKey := key + ":" + requestType
	
	// Get bucket or create if not exists
	bucket, exists := r.buckets[bucketKey]
	if !exists {
		// Get limit for this request type
		limit, ok := r.config.Limits[requestType]
		if !ok {
			limit = r.config.DefaultLimit
		}
		
		bucket = &tokenBucket{
			tokens:    limit,
			lastReset: time.Now(),
			limit:     limit,
		}
		r.buckets[bucketKey] = bucket
	}
	
	// Check if bucket should be refilled
	now := time.Now()
	elapsed := now.Sub(bucket.lastReset)
	if elapsed >= r.config.WindowSize {
		// Reset bucket
		bucket.tokens = bucket.limit
		bucket.lastReset = now
	}
	
	// Check if request can be allowed
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	
	return false
}

// cleanupLoop periodically cleans up expired buckets
func (r *TokenBucketRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(r.config.WindowSize * 2)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-r.done:
			return
		}
	}
}

// cleanup removes expired buckets
func (r *TokenBucketRateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	expiredTime := now.Add(-r.config.WindowSize * 3) // Remove buckets older than 3 windows
	
	for key, bucket := range r.buckets {
		if bucket.lastReset.Before(expiredTime) {
			delete(r.buckets, key)
		}
	}
}

// Stop stops the rate limiter and its background goroutines
func (r *TokenBucketRateLimiter) Stop() {
	close(r.done)
}