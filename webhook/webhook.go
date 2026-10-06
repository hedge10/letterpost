package webhook

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Event string

const (
	Before  Event = "before"
	After   Event = "after"
	Error   Event = "error"
	Success Event = "success"
)

func (e Event) valid() bool {
	switch e {
	case Before, After, Error, Success:
		return true
	}
	return false
}

type Webhook struct {
	Name   string  `json:"name"`
	URL    string  `json:"url"`
	Method string  `json:"method"`
	Events []Event `json:"events"`
}

func Load(path string) ([]Webhook, error) {
	data, err := os.ReadFile(path)
	// No webhooks.json is not an error
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var webhooks []Webhook
	err = json.Unmarshal(data, &webhooks, json.RejectUnknownMembers(true))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	for i := range webhooks {
		err := webhooks[i].validate()
		if err != nil {
			return nil, fmt.Errorf("%s: webhook %d: %w", path, i, err)
		}
	}

	return webhooks, nil
}

func (w *Webhook) validate() error {
	if strings.TrimSpace(w.Name) == "" {
		return errors.New("name is required")
	}

	u, err := url.Parse(w.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("url must be an absolute http(s) URL")
	}

	w.Method = strings.ToUpper(w.Method)
	switch w.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return errors.New("method must be POST, PUT or PATCH")
	}

	if len(w.Events) == 0 {
		return errors.New("events must not be empty")
	}
	for _, e := range w.Events {
		if !e.valid() {
			return fmt.Errorf("unknown event %q", e)
		}
	}

	return nil
}
