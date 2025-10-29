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

	sender := "test@example.com"

	// should allow burst requests immediately
	for i := range 5 {
		if !rl.allow(sender) {
			t.Errorf("request %d should be allowed (within burst)", i+1)
		}
	}

	// next request should be denied (burst exhausted)
	if rl.allow(sender) {
		t.Error("request should be denied after burst exhausted")
	}
}

func TestRateLimiterMultipleSenders(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// create rate limiter with 10 messages per minute, burst of 3
	rl := newRateLimiter(10, 3)
	rl.start(ctx)

	sender1 := "admin@company1"
	sender2 := "admin@company2"

	// each sender should have independent rate limits
	for i := range 3 {
		if !rl.allow(sender1) {
			t.Errorf("sender1 request %d should be allowed", i+1)
		}
		if !rl.allow(sender2) {
			t.Errorf("sender2 request %d should be allowed", i+1)
		}
	}

	// both senders should be rate limited now
	if rl.allow(sender1) {
		t.Error("sender1 should be rate limited")
	}
	if rl.allow(sender2) {
		t.Error("sender2 should be rate limited")
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

	sender := "test@example.com"

	// create a bucket
	rl.allow(sender)

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

	// test concurrent access to different senders
	done := make(chan bool)

	for i := range 10 {
		go func(sender string) {
			for range 100 {
				rl.allow(sender)
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
