package captcha

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		secret   string
		wantErr  bool
	}{
		{name: "unknown provider", provider: "recaptcha", secret: "secret", wantErr: true},
		{name: "empty provider", provider: "", secret: "secret", wantErr: true},
		{name: "empty secret with turnstile", provider: ProviderTurnstile, secret: "", wantErr: true},
		{name: "empty secret with friendly captcha", provider: ProviderFriendlyCaptcha, secret: "", wantErr: true},
		{name: "valid turnstile", provider: ProviderTurnstile, secret: "secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := New(tt.provider, tt.secret, "")

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if v == nil {
					t.Fatal("expected a Verifier, got nil")
				}
				return
			}

			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			// the middleware treats a nil Verifier as "captcha disabled",
			// so a failed New must never return a non-nil one
			if v != nil {
				t.Errorf("expected nil Verifier on error, got %T", v)
			}
		})
	}
}
