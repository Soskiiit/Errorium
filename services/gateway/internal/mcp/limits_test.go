package mcp

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestExtractIP(t *testing.T) {
	t.Run("X-Real-IP header present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("X-Real-IP", "203.0.113.195")
		req.RemoteAddr = "192.168.1.10:54321"

		ip := ExtractIP(req)
		if ip != "203.0.113.195" {
			t.Fatalf("expected 203.0.113.195, got %q", ip)
		}
	})

	t.Run("fallback to RemoteAddr host", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		req.RemoteAddr = "192.168.1.10:54321"

		ip := ExtractIP(req)
		if ip != "192.168.1.10" {
			t.Fatalf("expected 192.168.1.10, got %q", ip)
		}
	})

	t.Run("fallback to RemoteAddr without port", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		req.RemoteAddr = "192.168.1.10"

		ip := ExtractIP(req)
		if ip != "192.168.1.10" {
			t.Fatalf("expected 192.168.1.10, got %q", ip)
		}
	})
}

func TestLimiter_SessionLimits(t *testing.T) {
	// Configure limits: 2 sessions per IP, 3 sessions global
	cfg := LimiterConfig{
		MaxSessionsPerIP: 2,
		MaxSessions:      3,
		CallsPerMin:      10,
	}
	limiter := NewLimiter(cfg)

	// IP 1: Session 1
	allowed, isIPLimit := limiter.AllowSession("10.0.0.1")
	if !allowed {
		t.Fatalf("expected session 1 on 10.0.0.1 to be allowed")
	}
	if !limiter.RegisterSession("sess-1", "10.0.0.1") {
		t.Fatalf("failed to register sess-1")
	}

	// IP 1: Session 2
	allowed, isIPLimit = limiter.AllowSession("10.0.0.1")
	if !allowed {
		t.Fatalf("expected session 2 on 10.0.0.1 to be allowed")
	}
	if !limiter.RegisterSession("sess-2", "10.0.0.1") {
		t.Fatalf("failed to register sess-2")
	}

	// IP 1: Session 3 -> per-IP limit exceeded (429)
	allowed, isIPLimit = limiter.AllowSession("10.0.0.1")
	if allowed {
		t.Fatalf("expected session 3 on 10.0.0.1 to be rejected")
	}
	if !isIPLimit {
		t.Fatalf("expected isIPLimit=true for per-IP limit, got %v", isIPLimit)
	}

	// IP 2: Session 3 (global count is 2/3)
	allowed, isIPLimit = limiter.AllowSession("10.0.0.2")
	if !allowed {
		t.Fatalf("expected session 1 on 10.0.0.2 to be allowed")
	}
	if !limiter.RegisterSession("sess-3", "10.0.0.2") {
		t.Fatalf("failed to register sess-3")
	}

	// IP 3: Session 4 -> global ceiling reached (503)
	allowed, isIPLimit = limiter.AllowSession("10.0.0.3")
	if allowed {
		t.Fatalf("expected session on 10.0.0.3 to be rejected due to global ceiling")
	}
	if isIPLimit {
		t.Fatalf("expected isIPLimit=false for global limit, got %v", isIPLimit)
	}

	// Remove session 1 on IP 1 -> frees both per-IP slot and global slot
	limiter.RemoveSession("sess-1")

	// Now IP 1 can register again
	allowed, isIPLimit = limiter.AllowSession("10.0.0.1")
	if !allowed {
		t.Fatalf("expected session on 10.0.0.1 to be allowed after slot freed")
	}
	if !limiter.RegisterSession("sess-4", "10.0.0.1") {
		t.Fatalf("failed to register sess-4")
	}
}

func TestLimiter_CallLimitsSlidingWindow(t *testing.T) {
	// Configure limits: 2 calls per minute
	cfg := LimiterConfig{
		MaxSessionsPerIP: 10,
		MaxSessions:      20,
		CallsPerMin:      2,
	}
	limiter := NewLimiter(cfg)

	currTime := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	limiter.nowFunc = func() time.Time {
		return currTime
	}

	limiter.RegisterSession("sess-call", "10.0.0.1")

	// Call 1: Allowed
	if !limiter.AllowCall("sess-call") {
		t.Fatalf("call 1 should be allowed")
	}

	// Call 2: Allowed (total 2 in current minute)
	currTime = currTime.Add(10 * time.Second)
	if !limiter.AllowCall("sess-call") {
		t.Fatalf("call 2 should be allowed")
	}

	// Call 3: Rejected (2 calls already in sliding window)
	currTime = currTime.Add(10 * time.Second)
	if limiter.AllowCall("sess-call") {
		t.Fatalf("call 3 should be rejected by rate limit")
	}

	// Advance time by 45 seconds (total 65s since call 1)
	// Call 1 at +0s is now outside the 60s sliding window!
	currTime = currTime.Add(45 * time.Second)
	if !limiter.AllowCall("sess-call") {
		t.Fatalf("call 4 should be allowed as call 1 slid out of window")
	}

	// Next immediate call should be rejected
	if limiter.AllowCall("sess-call") {
		t.Fatalf("call 5 should be rejected")
	}
}

func TestLimiter_UnknownSession(t *testing.T) {
	limiter := NewLimiter(LimiterConfig{CallsPerMin: 5})
	if limiter.AllowCall("non-existent-session") {
		t.Fatalf("expected AllowCall for non-existent session to return false")
	}
}

func TestLimiter_Concurrency(t *testing.T) {
	limiter := NewLimiter(LimiterConfig{
		MaxSessionsPerIP: 50,
		MaxSessions:      100,
		CallsPerMin:      50,
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := "10.0.0.1"
			sessID := "sess-" + string(rune('A'+id))
			allowed, _ := limiter.AllowSession(ip)
			if allowed {
				limiter.RegisterSession(sessID, ip)
				limiter.AllowCall(sessID)
				limiter.RemoveSession(sessID)
			}
		}(i)
	}
	wg.Wait()
}
