package webhook

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"
)

type metadata struct {
	EventName   Event  `json:"event_name"`
	WebhookName string `json:"webhook_name"`
}

type Dispatcher struct {
	webhooks []Webhook
	client   *http.Client
	logger   *slog.Logger
	run      func(func())
}

// New returns a Dispatcher that sends each webhook request via run, which
// must execute the function in the background. A nil transport uses
// http.DefaultTransport.
func New(webhooks []Webhook, logger *slog.Logger, run func(func()), transport http.RoundTripper) *Dispatcher {
	return &Dispatcher{
		webhooks: webhooks,
		client: &http.Client{
			Transport: transport,
			Timeout:   3 * time.Second,
			// A redirect would turn the request into a GET without a body,
			// so it is reported as a failure instead.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		logger: logger,
		run:    run,
	}
}

// Fire notifies every webhook registered for event. The body holds fields
// in a "payload" member, next to a "metadata" member.
// Fire does not wait for the requests, failures are only logged.
func (d *Dispatcher) Fire(event Event, fields any) {
	for _, w := range d.webhooks {
		if !slices.Contains(w.Events, event) {
			continue
		}

		body, err := payload(fields, metadata{EventName: event, WebhookName: w.Name})
		if err != nil {
			d.fail(w, event, err)
			continue
		}

		d.run(func() { d.send(w, event, body) })
	}
}

func payload(fields any, meta metadata) ([]byte, error) {
	body := struct {
		Payload  any      `json:"payload"`
		Metadata metadata `json:"metadata"`
	}{Payload: fields, Metadata: meta}

	// Form values may not be valid UTF-8, invalid bytes become U+FFFD.
	return json.Marshal(body, jsontext.AllowInvalidUTF8(true))
}

func (d *Dispatcher) send(w Webhook, event Event, body []byte) {
	req, err := http.NewRequest(w.Method, w.URL, bytes.NewReader(body))
	if err != nil {
		d.fail(w, event, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		d.fail(w, event, err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		d.fail(w, event, fmt.Errorf("unexpected status %d", resp.StatusCode))
	}
}

// fail logs a failed webhook. Webhook URLs often carry secrets in the
// userinfo, path or query, so only the host is logged.
func (d *Dispatcher) fail(w Webhook, event Event, err error) {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		err = urlErr.Err
	}

	var host string
	if u, perr := url.Parse(w.URL); perr == nil {
		host = u.Host
	}

	d.logger.Error("webhook failed", "webhook", w.Name, "host", host, "event", event, "error", err.Error())
}
