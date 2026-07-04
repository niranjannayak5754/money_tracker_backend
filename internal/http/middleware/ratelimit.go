package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/niranjannayak5754/money_tracker_backend/internal/http/response"
)

// ipRateLimiter tracks one token-bucket limiter per client IP. Intended for
// low-traffic, single-instance deployments (no shared state needed); the map
// grows with the number of distinct IPs seen but that's acceptable at this
// app's scale.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	b        int
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	lim, ok := l.limiters[ip]
	if !ok {
		lim = rate.NewLimiter(l.r, l.b)
		l.limiters[ip] = lim
	}
	return lim
}

// RateLimit throttles requests per client IP to `r` events/sec with burst
// `b`. Meant for sensitive unauthenticated endpoints (login/register) to
// blunt brute-force/credential-stuffing attempts.
func RateLimit(r rate.Limit, b int) func(http.Handler) http.Handler {
	limiter := &ipRateLimiter{limiters: make(map[string]*rate.Limiter), r: r, b: b}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ip := clientIP(req)
			if !limiter.getLimiter(ip).Allow() {
				response.JSON(w, http.StatusTooManyRequests, map[string]string{
					"message": "too many requests, please try again later",
				})
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
