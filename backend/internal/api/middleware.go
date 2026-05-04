package api

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

type IPLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter // right now the bucket per ip will go on increasing a lot, we can somehow sweep the bucket if none requests are receieved in an interval(lets say 10 mins)
	rps      rate.Limit               // refill rate
	burst    int                      // bucket capactiy
}

func NewIPLimiter(rps float64, burst int) *IPLimiter {
	return &IPLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (l *IPLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.limiters[ip]; ok {
		return lim
	}
	lim := rate.NewLimiter(l.rps, l.burst)
	l.limiters[ip] = lim
	return lim
}

func (l *IPLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.get(ip).Allow() {
			w.Header().Set("Retry-After", "1") // can increase depending on use case and the app
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// decipher client ip
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if before, _, found := strings.Cut(xff, ","); found {
			return strings.TrimSpace(before)
		}
		return strings.TrimSpace(xff)
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
