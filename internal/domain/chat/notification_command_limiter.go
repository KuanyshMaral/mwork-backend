package chat

import (
	"sync"
	"time"
)

type userWindowLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	calls  map[string][]time.Time
}

func newUserWindowLimiter(limit int, window time.Duration) *userWindowLimiter {
	return &userWindowLimiter{
		limit:  limit,
		window: window,
		calls:  make(map[string][]time.Time),
	}
}

func (l *userWindowLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamps := l.calls[key]
	kept := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= l.limit {
		l.calls[key] = kept
		return false
	}

	kept = append(kept, now)
	l.calls[key] = kept
	return true
}
