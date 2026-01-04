package proxy

import (
	"sync"
	"time"
)

// RateLimiter implements the Token Bucket algorithm
type RateLimiter struct {
	rate       float64 // tokens per second
	capacity   float64 // max tokens
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new Token Bucket limiter
func NewRateLimiter(rate, capacity float64) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastUpdate: time.Now(),
	}
}

// Allow checks if a request can be proceeded
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// Refill tokens based on time passed
	elapsed := now.Sub(rl.lastUpdate).Seconds()
	rl.tokens += elapsed * rl.rate
	if rl.tokens > rl.capacity {
		rl.tokens = rl.capacity
	}
	rl.lastUpdate = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}

	return false
}
