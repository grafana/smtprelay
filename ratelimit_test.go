package main

import (
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// create rate limiter with 10 messages per minute, burst of 5
	rl := newRateLimiter(10, 5)
	rl.start(ctx)

	slug := "test@example.com"

	// should allow burst requests immediately
	for i := range 5 {
		if !rl.allow(slug) {
			t.Errorf("request %d should be allowed (within burst)", i+1)
		}
	}

	// next request should be denied (burst exhausted)
	if rl.allow(slug) {
		t.Error("request should be denied after burst exhausted")
	}
}

func TestRateLimiterMultipleSlugs(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// create rate limiter with 10 messages per minute, burst of 3
	rl := newRateLimiter(10, 3)
	rl.start(ctx)

	slug1 := "admin@company1"
	slug2 := "admin@company2"

	// each slug should have independent rate limits
	for i := range 3 {
		if !rl.allow(slug1) {
			t.Errorf("slug1 request %d should be allowed", i+1)
		}
		if !rl.allow(slug2) {
			t.Errorf("slug2 request %d should be allowed", i+1)
		}
	}

	// both slugs should be rate limited now
	if rl.allow(slug1) {
		t.Error("slug1 should be rate limited")
	}
	if rl.allow(slug2) {
		t.Error("slug2 should be rate limited")
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// create rate limiter with short TTL and cleanup interval for testing
	rl := newRateLimiter(10, 5)
	rl.bucketTTL = 50 * time.Millisecond
	rl.cleanupInterval = 100 * time.Millisecond
	rl.start(ctx)

	slug := "test@example.com"

	// create a bucket
	rl.allow(slug)

	// check bucket exists
	rl.mu.Lock()
	if len(rl.limiters) != 1 {
		t.Errorf("expected 1 bucket, got %d", len(rl.limiters))
	}
	rl.mu.Unlock()

	// wait for cleanup to remove inactive bucket
	time.Sleep(200 * time.Millisecond)

	rl.mu.Lock()
	bucketCount := len(rl.limiters)
	rl.mu.Unlock()

	if bucketCount != 0 {
		t.Errorf("expected 0 buckets after cleanup, got %d", bucketCount)
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	rl := newRateLimiter(100, 10)

	rl.start(ctx)

	// test concurrent access to different slugs
	done := make(chan bool)

	for i := range 10 {
		go func(slug string) {
			for range 100 {
				rl.allow(slug)
			}
			done <- true
		}(string(rune('a' + i)))
	}

	// wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// check that all buckets were created
	rl.mu.Lock()
	bucketCount := len(rl.limiters)
	rl.mu.Unlock()

	if bucketCount != 10 {
		t.Errorf("expected 10 buckets, got %d", bucketCount)
	}
}
