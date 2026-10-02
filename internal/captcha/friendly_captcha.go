package captcha

import (
	"context"
	"fmt"
	"time"

	frc "github.com/friendlycaptcha/friendly-captcha-go"
)

type friendlyCaptcha struct {
	client *frc.Client
}

func newFriendlyCaptcha(apiKey, sitekey string) (*friendlyCaptcha, error) {
	opts := []frc.ClientOption{
		frc.WithAPIKey(apiKey),
		frc.WithStrictMode(true),
	}
	if sitekey != "" {
		opts = append(opts, frc.WithSitekey(sitekey))
	}

	client, err := frc.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &friendlyCaptcha{client: client}, nil
}

func (f *friendlyCaptcha) FormField() string {
	return frc.ResponseFormFieldName // "frc-captcha-response"
}

func (f *friendlyCaptcha) Verify(ctx context.Context, token, _ string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result := f.client.VerifyCaptchaResponse(ctx, token)

	// network problem, wrong API key or sitekey: not the visitor's fault
	if !result.WasAbleToVerify() {
		if err := result.RequestError(); err != nil {
			return fmt.Errorf("friendly captcha: %w", err)
		}
		return fmt.Errorf("friendly captcha: verification failed with HTTP %d", result.HTTPStatusCode())
	}

	if result.ShouldReject() {
		return ErrInvalidToken
	}

	return nil
}
