package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// ipRateLimiter is a simple in-memory sliding-window rate limiter keyed by
// client IP address.
type ipRateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// newIPRateLimiter returns a rate limiter that allows at most limit requests
// per window for each IP address.
func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// allow reports whether a request from ip should be permitted, recording it
// if so.
func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	kept := l.requests[ip][:0]
	for _, t := range l.requests[ip] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.limit {
		l.requests[ip] = kept
		return false
	}

	l.requests[ip] = append(kept, now)
	return true
}

// middleware returns an http.Handler middleware that rejects requests
// exceeding the rate limit with 429 Too Many Requests.
func (l *ipRateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}

		if !l.allow(ip) {
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}

		next.ServeHTTP(w, r)
	})
}
