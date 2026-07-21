package main

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type fixedWindowEntry struct {
	windowStart time.Time
	count       int
	lastSeen    time.Time
}

type fixedWindowRateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	maxHits int
	entries map[string]fixedWindowEntry
}

func newFixedWindowRateLimiter(window time.Duration, maxHits int) *fixedWindowRateLimiter {
	return &fixedWindowRateLimiter{
		window:  window,
		maxHits: maxHits,
		entries: make(map[string]fixedWindowEntry),
	}
}

// method für fixedWindowRateLimiter returns true wenn die angegebene maximale refresh/login limitirung noch nicht überschritten wurde
func (l *fixedWindowRateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entries[key]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= l.window {
		entry.windowStart = now
		entry.count = 0
	}

	entry.count++
	entry.lastSeen = now
	l.entries[key] = entry

	// Garbage clean für zu großen lokalen speicher
	if len(l.entries) > 20000 {
		cutoff := now.Add(-2 * l.window)
		for k, v := range l.entries {
			if v.lastSeen.Before(cutoff) {
				delete(l.entries, k)
			}
		}
	}

	return entry.count <= l.maxHits
}

func (l *fixedWindowRateLimiter) isAtOrOverLimit(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.entries[key]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= l.window {
		return false
	}

	return entry.count >= l.maxHits
}

func (l *fixedWindowRateLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// middleware für refresh/login limit per IP
func rateLimitByIP(routeName string, limiter *fixedWindowRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		key := routeName + ":" + clientIP
		if !limiter.allow(key, time.Now().UTC()) {
			respondWithError(w, http.StatusTooManyRequests, "Too many requests, please try again later.", nil)
			return
		}

		next(w, r)
	}
}

type statusCaptureResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCaptureResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// middleware für fehlgeschlagene logins
func rateLimitFailedLoginsByIP(routeName string, failureLimiter *fixedWindowRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		key := routeName + ":" + clientIP
		now := time.Now().UTC()

		if failureLimiter.isAtOrOverLimit(key, now) {
			respondWithError(w, http.StatusTooManyRequests, "Too many failed login attempts, please try again later.", nil)
			return
		}

		statusWriter := &statusCaptureResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(statusWriter, r)

		if statusWriter.statusCode == http.StatusUnauthorized {
			failureLimiter.allow(key, time.Now().UTC())
			return
		}

		if statusWriter.statusCode >= 200 && statusWriter.statusCode < 400 {
			failureLimiter.reset(key)
		}
	}
}

func getClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			candidate := strings.TrimSpace(parts[0])
			if candidate != "" {
				return candidate
			}
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	if strings.TrimSpace(r.RemoteAddr) != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}

	return "unknown"
}
