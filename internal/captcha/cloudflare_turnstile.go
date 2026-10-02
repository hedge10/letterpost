package captcha

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const turnstileSiteVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type turnstile struct {
	secret string
	client *http.Client
}

type turnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTs string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
}

var turnstileConfigErrors = []string{
	"missing-input-secret",
	"invalid-input-secret",
	"bad-request",
	"internal-error",
}

func newTurnstile(secret string) *turnstile {
	return &turnstile{
		secret: secret,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (t *turnstile) FormField() string {
	return "cf-turnstile-response"
}

func (t *turnstile) Verify(ctx context.Context, token, remoteIP string) error {
	form := url.Values{}
	form.Set("secret", t.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileSiteVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("turnstile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("turnstile: siteverify returned HTTP %s", resp.Status)
	}

	var result turnstileResponse
	err = json.UnmarshalRead(resp.Body, &result)
	if err != nil {
		return fmt.Errorf("turnstile: decoding response: %w", err)
	}

	if result.Success {
		return nil
	}

	for _, code := range result.ErrorCodes {
		if slices.Contains(turnstileConfigErrors, code) {
			return fmt.Errorf("turnstile: %v", result.ErrorCodes)
		}
	}

	// invalid-input-response, timeout-or-duplicate, missing-input-response, ...
	return fmt.Errorf("%w: %v", ErrInvalidToken, result.ErrorCodes)
}
