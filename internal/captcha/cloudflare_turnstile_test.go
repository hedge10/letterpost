package captcha

import (
	"context"
	"errors"
	"testing"
)

// Cloudflare's test secrets behave predictably with any token.
// See https://developers.cloudflare.com/turnstile/troubleshooting/testing/
const (
	turnstileSecretPass  = "1x0000000000000000000000000000000AA"
	turnstileSecretFail  = "2x0000000000000000000000000000000AA"
	turnstileSecretSpent = "3x0000000000000000000000000000000AA"

	// token produced by the test sitekeys
	turnstileDummyToken = "XXXX.DUMMY.TOKEN.XXXX"
)

func TestTurnstileVerify(t *testing.T) {
	tests := []struct {
		name        string
		secret      string
		wantErr     bool
		wantInvalid bool // error must wrap ErrInvalidToken
	}{
		{name: "token passes", secret: turnstileSecretPass},
		{name: "token fails", secret: turnstileSecretFail, wantErr: true, wantInvalid: true},
		{name: "token already spent", secret: turnstileSecretSpent, wantErr: true, wantInvalid: true},
		{name: "invalid secret is a config error", secret: "invalid-secret", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newTurnstile(tt.secret).Verify(context.Background(), turnstileDummyToken, "")

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if got := errors.Is(err, ErrInvalidToken); got != tt.wantInvalid {
				t.Errorf("errors.Is(err, ErrInvalidToken) = %v, want %v (err: %v)", got, tt.wantInvalid, err)
			}
		})
	}
}
