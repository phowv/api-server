package ratelimiter

import (
	"sync"
	"time"
)

type InMemoryRateLimiter struct {
	visits map[string][]time.Time
	mx sync.Mutex
}

func NewInMemoryRateLimiter() *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		visits: make(map[string][]time.Time),
	}
}

func (r *InMemoryRateLimiter) Allow(key string, limit int, window time.Duration) bool {
	now := time.Now()

	r.mx.Lock()
	times := r.visits[key]

	j := 0
	for _, visitTime := range times {
		if now.Sub(visitTime) <= window {
			times[j] = visitTime
			j++
		}
	}

	times = times[:j]

	times = append(times, now)

	r.visits[key] = times

	allows := len(times) <= limit

	r.mx.Unlock()

	return allows
}
