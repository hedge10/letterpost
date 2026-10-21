package middleware

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		origin        string
		requestMethod string
		wantOrigin    string
		wantMethods   bool
		wantStatus    int
		wantNext      bool
	}{
		{name: "trusted origin", method: http.MethodPost, origin: "https://example.com", wantOrigin: "https://example.com", wantStatus: http.StatusTeapot, wantNext: true},
		{name: "untrusted origin", method: http.MethodPost, origin: "https://evil.example", wantStatus: http.StatusTeapot, wantNext: true},
		{name: "no origin", method: http.MethodPost, wantStatus: http.StatusTeapot, wantNext: true},
		{name: "preflight from trusted origin", method: http.MethodOptions, origin: "https://example.com", requestMethod: http.MethodPost, wantOrigin: "https://example.com", wantMethods: true, wantStatus: http.StatusOK},
		{name: "preflight from untrusted origin", method: http.MethodOptions, origin: "https://evil.example", requestMethod: http.MethodPost, wantStatus: http.StatusTeapot, wantNext: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusTeapot)
			})
			h := CORS([]string{"https://example.com"}, next)

			r := httptest.NewRequest(tt.method, "/", nil)
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			if tt.requestMethod != "" {
				r.Header.Set("Access-Control-Request-Method", tt.requestMethod)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if called != tt.wantNext {
				t.Errorf("next called = %v, want %v", called, tt.wantNext)
			}
			if got := rr.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantOrigin)
			}
			if got := rr.Header().Get("Access-Control-Allow-Methods") != ""; got != tt.wantMethods {
				t.Errorf("Access-Control-Allow-Methods set = %v, want %v", got, tt.wantMethods)
			}
		})
	}
}

func TestCORSVary(t *testing.T) {
	h := CORS(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", nil))

	vary := rr.Header().Values("Vary")
	for _, want := range []string{"Origin", "Access-Control-Request-Method"} {
		if !slices.Contains(vary, want) {
			t.Errorf("Vary = %q, want it to contain %q", vary, want)
		}
	}
}
