package captcha

import (
	"context"
	"errors"
	"fmt"
)

type Provider string

const (
	ProviderTurnstile       Provider = "cloudflare"
	ProviderFriendlyCaptcha Provider = "friendly-captcha"
)

// ErrInvalidToken is returned by Verify when the provider rejected the token
// (missing, expired, already used, wrong sitekey, ...). Any other error means
// the token could not be checked at all, e.g. the provider was unreachable.
var ErrInvalidToken = errors.New("captcha token is invalid")

// Verifier checks a captcha token from a submitted form with its provider.
type Verifier interface {
	// FormField is the name of the form field the widget puts its token in.
	FormField() string
	// Verify checks the token. It returns nil if the token is valid, an error
	// wrapping ErrInvalidToken if the provider rejected it, or any other error
	// if verification itself failed. remoteIP may be empty.
	Verify(ctx context.Context, token, remoteIP string) error
}

func New(provider Provider, secret, sitekey string) (Verifier, error) {
	if secret == "" {
		return nil, errors.New("captcha secret is required")
	}

	switch provider {
	case ProviderTurnstile:
		return newTurnstile(secret), nil
	case ProviderFriendlyCaptcha:
		v, err := newFriendlyCaptcha(secret, sitekey)
		if err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, fmt.Errorf("invalid captcha provider %q (%s|%s)", provider, ProviderTurnstile, ProviderFriendlyCaptcha)
	}
}
