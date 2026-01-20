package proxy

import (
	"testing"
	"time"
)

func TestRateLimiterInitialBurst(t *testing.T) {
	rl := NewRateLimiter(1, 5) // 1 token/second, capacity 5

	// At start, bucket is full, so we can perform 5 successful requests in a row.
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Fatalf("expected Allow to return true on initial burst, iteration %d", i)
		}
	}

	// Next call should be blocked because there are no tokens left.
	if rl.Allow() {
		t.Fatalf("expected Allow to return false when bucket is empty")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	rl := NewRateLimiter(2, 2) // 2 tokens/second, capacity 2

	// Consume both tokens.
	if !rl.Allow() || !rl.Allow() {
		t.Fatalf("expected initial two Allows to succeed")
	}

	if rl.Allow() {
		t.Fatalf("expected third immediate Allow to fail when bucket is empty")
	}

	// Wait slightly more than half a second so at least one token is refilled.
	time.Sleep(600 * time.Millisecond)

	if !rl.Allow() {
		t.Fatalf("expected Allow to succeed after tokens refill")
	}
}
