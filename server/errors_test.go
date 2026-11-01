package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"codeberg.org/hedge10/staticform/captcha"
)

type fakeVerifier struct{ err error }

func (f fakeVerifier) FormField() string { return "token" }

func (f fakeVerifier) Verify(ctx context.Context, token, remoteIP string) error { return f.err }

func TestFormErrorRedirect(t *testing.T) {
	invalidForm := url.Values{"sender": {"not-an-address"}}

	tests := []struct {
		name         string
		redirectURL  string
		form         url.Values
		verifyErr    error
		useCaptcha   bool
		wantStatus   int
		wantLocation string
		wantPrefix   string
		wantSuffix   string
		wantLog      string
		wantNoLog    string
	}{
		{
			name:         "validation error redirects with error code",
			redirectURL:  "https://example.com/thanks",
			form:         invalidForm,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "https://example.com/thanks?error=invalid",
		},
		{
			name:         "_redirect_error picks the target",
			redirectURL:  "https://example.com/thanks",
			form:         url.Values{"sender": {"not-an-address"}, "_redirect_error": {"/oops"}},
			wantStatus:   http.StatusSeeOther,
			wantLocation: "https://example.com/oops?error=invalid",
		},
		{
			name:        "_redirect_error cannot leave the origin",
			redirectURL: "https://example.com/thanks",
			form:        url.Values{"sender": {"not-an-address"}, "_redirect_error": {"//evil.example"}},
			wantStatus:  http.StatusSeeOther,
			wantPrefix:  "https://example.com/",
			wantSuffix:  "error=invalid",
		},
		{
			name:         "unparsable _redirect_error falls back to the base",
			redirectURL:  "https://example.com/thanks",
			form:         url.Values{"sender": {"not-an-address"}, "_redirect_error": {"/x%zz"}},
			wantStatus:   http.StatusSeeOther,
			wantLocation: "https://example.com/thanks?error=invalid",
			wantNoLog:    "SF_REDIRECT_URL",
		},
		{
			name:       "no redirect url keeps JSON",
			form:       invalidForm,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "_redirect_error without redirect url is logged",
			form:       url.Values{"sender": {"not-an-address"}, "_redirect_error": {"/oops"}},
			wantStatus: http.StatusUnprocessableEntity,
			wantLog:    "SF_REDIRECT_URL",
		},
		{
			name:         "missing captcha token",
			redirectURL:  "https://example.com/thanks",
			form:         url.Values{},
			useCaptcha:   true,
			verifyErr:    errors.New("unused"),
			wantStatus:   http.StatusSeeOther,
			wantLocation: "https://example.com/thanks?error=captcha_missing",
		},
		{
			name:         "rejected captcha token",
			redirectURL:  "https://example.com/thanks",
			form:         url.Values{"token": {"abc"}},
			useCaptcha:   true,
			verifyErr:    captcha.ErrInvalidToken,
			wantStatus:   http.StatusSeeOther,
			wantLocation: "https://example.com/thanks?error=captcha_failed",
		},
		{
			name:         "captcha provider failure",
			redirectURL:  "https://example.com/thanks",
			form:         url.Values{"token": {"abc"}},
			useCaptcha:   true,
			verifyErr:    errors.New("provider down"),
			wantStatus:   http.StatusSeeOther,
			wantLocation: "https://example.com/thanks?error=captcha_unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			app := &Server{logger: slog.New(slog.NewTextHandler(&logs, nil))}
			app.config.RedirectURL = tt.redirectURL

			handler := http.HandlerFunc(app.sendMail)
			if tt.useCaptcha {
				app.captchaVerifier = fakeVerifier{err: tt.verifyErr}
				handler = app.captcha(http.NotFoundHandler()).ServeHTTP
			}

			r := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(tt.form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			loc := w.Header().Get("Location")
			if tt.wantLocation != "" && loc != tt.wantLocation {
				t.Errorf("Location = %q, want %q", loc, tt.wantLocation)
			}
			if tt.wantPrefix != "" && !strings.HasPrefix(loc, tt.wantPrefix) {
				t.Errorf("Location = %q, want prefix %q", loc, tt.wantPrefix)
			}
			if tt.wantSuffix != "" && !strings.HasSuffix(loc, tt.wantSuffix) {
				t.Errorf("Location = %q, want suffix %q", loc, tt.wantSuffix)
			}
			if tt.wantStatus != http.StatusSeeOther && loc != "" {
				t.Errorf("unexpected Location %q", loc)
			}
			if tt.wantNoLog != "" && strings.Contains(logs.String(), tt.wantNoLog) {
				t.Errorf("log %q contains %q", logs.String(), tt.wantNoLog)
			}
			if tt.wantLog != "" && !strings.Contains(logs.String(), tt.wantLog) {
				t.Errorf("log %q does not contain %q", logs.String(), tt.wantLog)
			}
		})
	}
}

func TestNotFoundIsNeverRedirected(t *testing.T) {
	app := &Server{logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))}
	app.config.RedirectURL = "https://example.com/thanks"

	r := httptest.NewRequest(http.MethodPost, "/nope", nil)
	w := httptest.NewRecorder()

	app.notFoundResponse(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("unexpected Location %q", loc)
	}
}

func TestMethodNotAllowedIsNeverRedirected(t *testing.T) {
	app := &Server{logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))}
	app.config.RedirectURL = "https://example.com/thanks"

	r := httptest.NewRequest(http.MethodDelete, "/v1/send", nil)
	w := httptest.NewRecorder()

	app.methodNotAllowedResponse(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	if loc := w.Header().Get("Location"); loc != "" {
		t.Errorf("unexpected Location %q", loc)
	}
}
