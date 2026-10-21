package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// limited returns a handler limited by cfg that answers 200, or 429 when
// the limit is exceeded.
func limited(cfg RateLimitConfig) http.Handler {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	exceeded := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}
	return RateLimit(cfg, next, exceeded)
}

// sendFrom sends a GET request from remoteAddr, with forwardedFor as the
// X-Forwarded-For header unless it is empty, and returns the status.
func sendFrom(h http.Handler, remoteAddr, forwardedFor string) int {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		r.Header.Set("X-Forwarded-For", forwardedFor)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr.Code
}

func TestRateLimit(t *testing.T) {
	h := limited(RateLimitConfig{RPS: 0.001, Burst: 2, Enabled: true})

	for i := range 2 {
		if got := sendFrom(h, "192.0.2.1:1234", ""); got != http.StatusOK {
			t.Fatalf("request %d: got status %d; want %d", i+1, got, http.StatusOK)
		}
	}
	if got := sendFrom(h, "192.0.2.1:1234", ""); got != http.StatusTooManyRequests {
		t.Errorf("got status %d; want %d", got, http.StatusTooManyRequests)
	}
	if got := sendFrom(h, "192.0.2.2:1234", ""); got != http.StatusOK {
		t.Errorf("other IP: got status %d; want %d", got, http.StatusOK)
	}
}

func TestRateLimitUsesForwardedFor(t *testing.T) {
	h := limited(RateLimitConfig{RPS: 0.001, Burst: 1, Enabled: true})

	if got := sendFrom(h, "10.0.0.1:1234", "198.51.100.7"); got != http.StatusOK {
		t.Fatalf("got status %d; want %d", got, http.StatusOK)
	}
	if got := sendFrom(h, "10.0.0.1:1234", "198.51.100.8"); got != http.StatusOK {
		t.Errorf("other client: got status %d; want %d", got, http.StatusOK)
	}
	if got := sendFrom(h, "10.0.0.1:1234", "198.51.100.7"); got != http.StatusTooManyRequests {
		t.Errorf("same client: got status %d; want %d", got, http.StatusTooManyRequests)
	}
}

func TestRateLimitDisabled(t *testing.T) {
	h := limited(RateLimitConfig{RPS: 0.001, Burst: 1, Enabled: false})

	for i := range 3 {
		if got := sendFrom(h, "192.0.2.1:1234", ""); got != http.StatusOK {
			t.Fatalf("request %d: got status %d; want %d", i+1, got, http.StatusOK)
		}
	}
}
