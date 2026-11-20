package mcp

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LimiterConfig holds thresholds for MCP rate and session limits.
type LimiterConfig struct {
	MaxSessionsPerIP int
	MaxSessions      int
	CallsPerMin      int
}

// SessionRecord tracks active session metadata and tool call timestamps.
type SessionRecord struct {
	IP        string
	CreatedAt time.Time
	CallTimes []time.Time
}

// Limiter manages thread-safe in-memory session and call rate limits.
type Limiter struct {
	mu               sync.Mutex
	maxSessionsPerIP int
	maxSessions      int
	callsPerMin      int
	sessions         map[string]*SessionRecord
	nowFunc          func() time.Time
}

// NewLimiter creates a new Limiter with specified configuration thresholds.
func NewLimiter(cfg LimiterConfig) *Limiter {
	maxIP := cfg.MaxSessionsPerIP
	if maxIP <= 0 {
		maxIP = 100
	}
	maxGlobal := cfg.MaxSessions
	if maxGlobal <= 0 {
		maxGlobal = 200
	}
	calls := cfg.CallsPerMin
	if calls <= 0 {
		calls = 60
	}
	return &Limiter{
		maxSessionsPerIP: maxIP,
		maxSessions:      maxGlobal,
		callsPerMin:      calls,
		sessions:         make(map[string]*SessionRecord),
		nowFunc:          time.Now,
	}
}

// AllowSession checks whether a new session from ip is allowed.
// Returns allowed: true if allowed.
// If allowed is false, isIPLimit is true if the per-IP limit was exceeded,
// or false if the global session limit was exceeded.
func (l *Limiter) AllowSession(ip string) (bool, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	ipCount := 0
	for _, s := range l.sessions {
		if s.IP == ip {
			ipCount++
		}
	}

	if ipCount >= l.maxSessionsPerIP {
		return false, true
	}
	if len(l.sessions) >= l.maxSessions {
		return false, false
	}
	return true, false
}

// RegisterSession registers a new session in the limiter atomically.
func (l *Limiter) RegisterSession(sessionID, ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	ipCount := 0
	for _, s := range l.sessions {
		if s.IP == ip {
			ipCount++
		}
	}

	if ipCount >= l.maxSessionsPerIP || len(l.sessions) >= l.maxSessions {
		return false
	}

	l.sessions[sessionID] = &SessionRecord{
		IP:        ip,
		CreatedAt: l.nowFunc(),
		CallTimes: make([]time.Time, 0, 10),
	}
	return true
}

// HasSession checks if a session exists in the registry.
func (l *Limiter) HasSession(sessionID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, exists := l.sessions[sessionID]
	return exists
}

// RemoveSession deletes the session, releasing the slot.
func (l *Limiter) RemoveSession(sessionID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.sessions, sessionID)
}

// ActiveSessionsCount returns the total number of active sessions.
func (l *Limiter) ActiveSessionsCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.sessions)
}

// AllowCall checks if tools/call is allowed within the 60-second sliding window.
// If allowed, records the timestamp and returns true. If rate limit exceeded or session unknown, returns false.
func (l *Limiter) AllowCall(sessionID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	s, ok := l.sessions[sessionID]
	if !ok {
		return false
	}

	now := l.nowFunc()
	windowStart := now.Add(-60 * time.Second)

	// Clean up timestamps older than 60s
	validIdx := 0
	for i, t := range s.CallTimes {
		if t.After(windowStart) {
			validIdx = i
			break
		}
		if i == len(s.CallTimes)-1 {
			validIdx = len(s.CallTimes)
		}
	}
	if validIdx > 0 {
		s.CallTimes = s.CallTimes[validIdx:]
	}

	if len(s.CallTimes) >= l.callsPerMin {
		return false
	}

	s.CallTimes = append(s.CallTimes, now)
	return true
}

// ExtractIP extracts client IP from X-Real-IP header or fallback to host from RemoteAddr.
func ExtractIP(r *http.Request) string {
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return strings.TrimSpace(xRealIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
