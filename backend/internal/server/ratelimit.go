package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type multiLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	ttl     time.Duration
	entries map[string]*limBucket
}

type limBucket struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

func newMultiLimiter(limit rate.Limit, burst int, ttl time.Duration) *multiLimiter {
	return &multiLimiter{
		limit:   limit,
		burst:   burst,
		ttl:     ttl,
		entries: make(map[string]*limBucket),
	}
}

func (m *multiLimiter) allow(key string) bool {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.entries[key]
	if b == nil {
		b = &limBucket{lim: rate.NewLimiter(m.limit, m.burst), lastSeen: now}
		m.entries[key] = b
	}
	b.lastSeen = now

	for k, v := range m.entries {
		if now.Sub(v.lastSeen) > m.ttl {
			delete(m.entries, k)
		}
	}
	return b.lim.Allow()
}

func getClientIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
