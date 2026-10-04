package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRedirectTarget(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		field       string
		want        string
		wantErr     bool
	}{
		{name: "nothing configured, no field", want: ""},
		{name: "fallback to env var", redirectURL: "https://example.com/thanks", want: "https://example.com/thanks"},
		{name: "path is put on env var origin", redirectURL: "https://example.com/thanks", field: "/de/danke", want: "https://example.com/de/danke"},
		{name: "env var port is kept", redirectURL: "http://localhost:8080/thanks", field: "/other", want: "http://localhost:8080/other"},
		{name: "query and fragment are kept", redirectURL: "https://example.com/", field: "/thanks?form=contact#top", want: "https://example.com/thanks?form=contact#top"},
		{name: "path without leading slash", redirectURL: "https://example.com/", field: "de/danke", want: "https://example.com/de/danke"},
		{name: "protocol-relative url stays on origin", redirectURL: "https://example.com/", field: "//evil.com/", want: "https://example.com/evil.com/"},
		{name: "full url stays on origin", redirectURL: "https://example.com/", field: "https://evil.com/", want: "https://example.com/https://evil.com/"},
		{name: "path without env var", field: "/thanks", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &application{}
			app.config.RedirectURL = tt.redirectURL

			got, err := app.redirectTarget(tt.field)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got target %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("target = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSendMailRejectsRedirectWithoutRedirectURL(t *testing.T) {
	app := &application{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	form := url.Values{
		"name":       {"Jane Doe"},
		"sender":     {"jane.doe@gmail.com"},
		"plain_body": {"Hi"},
		"_redirect":  {"/thanks"},
	}
	r := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	app.sendMail(w, r)
	app.wg.Wait()

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
	if !strings.Contains(w.Body.String(), "_redirect") {
		t.Errorf("expected an error for _redirect, body: %s", w.Body.String())
	}
}

func TestHoneypotRedirects(t *testing.T) {
	app := &application{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	app.config.HoneypotField = "website"
	app.config.RedirectURL = "https://example.com/thanks"

	form := url.Values{"website": {"spam"}}
	r := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	app.honeypot(http.NotFoundHandler()).ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/thanks" {
		t.Errorf("Location = %q, want %q", loc, "https://example.com/thanks")
	}
}
