package main

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimiter manages per-sender rate limiting using token buckets
type rateLimiter struct {
	limiters map[string]*bucketEntry
	mu       sync.Mutex

	messagesPerSecond float64
	burst             int
	cleanupInterval   time.Duration
	bucketTTL         time.Duration
}

// bucketEntry holds a rate limiter and its last access time
type bucketEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// newRateLimiter creates a new rate limiter with the given configuration
func newRateLimiter(messagesPerSecond float64, burst int) *rateLimiter {
	return &rateLimiter{
		limiters:          make(map[string]*bucketEntry),
		messagesPerSecond: messagesPerSecond,
		burst:             burst,
		cleanupInterval:   15 * time.Minute,
		bucketTTL:         1 * time.Hour,
	}
}

// start kicks off
func (rl *rateLimiter) start(ctx context.Context) {
	go rl.cleanupLoop(ctx)
}

// allow checks if a request for the given sender should be allowed
func (rl *rateLimiter) allow(sender string) bool {
	return rl.getLimiter(sender).Allow()
}

// getLimiter returns the rate limiter for a given sender, creating one if needed
func (rl *rateLimiter) getLimiter(sender string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, exists := rl.limiters[sender]
	if exists {
		entry.lastAccess = time.Now()
		return entry.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rl.messagesPerSecond), rl.burst)

	rl.limiters[sender] = &bucketEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}

	return limiter
}

// cleanupLoop periodically removes unused rate limiters
func (rl *rateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-ctx.Done():
			return
		}
	}
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for sender, entry := range rl.limiters {
		if now.Sub(entry.lastAccess) > rl.bucketTTL {
			delete(rl.limiters, sender)
		}
	}
}
