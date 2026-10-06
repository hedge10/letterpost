package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHoneypot(t *testing.T) {
	tests := []struct {
		name       string
		field      string
		form       url.Values
		wantNext   bool
		wantStatus int
	}{
		{name: "disabled when no field is configured", field: "", form: url.Values{"website": {"spam"}}, wantNext: true, wantStatus: http.StatusTeapot},
		{name: "empty honeypot passes", field: "website", form: url.Values{"website": {""}}, wantNext: true, wantStatus: http.StatusTeapot},
		{name: "missing honeypot passes", field: "website", form: url.Values{}, wantNext: true, wantStatus: http.StatusTeapot},
		{name: "filled honeypot is blocked", field: "website", form: url.Values{"website": {"http://spam.example"}, "sender": {"bot@example.com"}}, wantNext: false, wantStatus: http.StatusOK},
		{name: "whitespace-only honeypot is blocked", field: "website", form: url.Values{"website": {" "}}, wantNext: false, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &Server{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			app.config.HoneypotField = tt.field

			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusTeapot)
			})

			r := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(tt.form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			app.honeypot(next).ServeHTTP(w, r)

			if called != tt.wantNext {
				t.Errorf("next handler called = %v, want %v", called, tt.wantNext)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if !tt.wantNext && !strings.Contains(w.Body.String(), `"email_status": "sent"`) {
				t.Errorf("blocked request should look like a successful send, body: %s", w.Body.String())
			}
		})
	}
}
