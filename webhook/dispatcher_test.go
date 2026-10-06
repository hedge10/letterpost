package webhook

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type message struct {
	Name      string `json:"name"`
	PlainBody string `json:"plain_body"`
}

type request struct {
	Method      string
	ContentType string
	Body        map[string]any
}

type recorder struct {
	mu       sync.Mutex
	requests []request
}

// newRecorder starts a server that records every request and answers with status.
func newRecorder(t *testing.T, status int) (*recorder, *httptest.Server) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.UnmarshalRead(r.Body, &body)

		rec.mu.Lock()
		rec.requests = append(rec.requests, request{Method: r.Method, ContentType: r.Header.Get("Content-Type"), Body: body})
		rec.mu.Unlock()

		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return rec, srv
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestFireSendsFieldsAndMetadata(t *testing.T) {
	rec, srv := newRecorder(t, http.StatusOK)
	var wg sync.WaitGroup
	d := New([]Webhook{{Name: "crm", URL: srv.URL, Method: http.MethodPut, Events: []Event{Success}}}, discardLogger(), wg.Go, nil)

	d.Fire(Success, message{Name: `Jürgen "JJ" Doe`, PlainBody: "Line one\nLine two"})
	wg.Wait()

	if len(rec.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(rec.requests))
	}
	got := rec.requests[0]
	if got.Method != http.MethodPut {
		t.Errorf("expected method PUT, got %s", got.Method)
	}
	if got.ContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", got.ContentType)
	}
	want := map[string]any{
		"payload": map[string]any{
			"name":       `Jürgen "JJ" Doe`,
			"plain_body": "Line one\nLine two",
		},
		"metadata": map[string]any{
			"event_name":   "success",
			"webhook_name": "crm",
		},
	}
	if !reflect.DeepEqual(got.Body, want) {
		t.Errorf("got body %v, want %v", got.Body, want)
	}
}

func TestFireOnlyMatchingWebhooks(t *testing.T) {
	rec, srv := newRecorder(t, http.StatusOK)
	var wg sync.WaitGroup
	d := New([]Webhook{
		{Name: "on-success", URL: srv.URL, Method: http.MethodPost, Events: []Event{Success}},
		{Name: "on-error", URL: srv.URL, Method: http.MethodPost, Events: []Event{Error}},
		{Name: "duplicate-event", URL: srv.URL, Method: http.MethodPost, Events: []Event{Success, Success}},
	}, discardLogger(), wg.Go, nil)

	d.Fire(Success, message{Name: "Jane Doe"})
	wg.Wait()

	var names []string
	for _, r := range rec.requests {
		names = append(names, r.Body["metadata"].(map[string]any)["webhook_name"].(string))
	}
	slices.Sort(names)
	want := []string{"duplicate-event", "on-success"}
	if !slices.Equal(names, want) {
		t.Errorf("called %v, want %v", names, want)
	}
}

func TestFireLogsFailures(t *testing.T) {
	_, failing := newRecorder(t, http.StatusInternalServerError)
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	tests := []struct {
		name string
		url  string
	}{
		{name: "non-2xx response", url: failing.URL},
		{name: "unreachable url", url: closed.URL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			var wg sync.WaitGroup
			d := New([]Webhook{{Name: "crm", URL: tt.url, Method: http.MethodPost, Events: []Event{Error}}}, slog.New(slog.NewTextHandler(&buf, nil)), wg.Go, nil)

			d.Fire(Error, message{Name: "Jane Doe"})
			wg.Wait()

			out := buf.String()
			if !strings.Contains(out, `msg="webhook failed"`) || !strings.Contains(out, "webhook=crm") {
				t.Errorf("expected the failure to be logged, got %q", out)
			}
		})
	}
}

func TestFireDoesNotWaitForSlowReceiver(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	var wg sync.WaitGroup
	d := New([]Webhook{{Name: "slow", URL: srv.URL, Method: http.MethodPost, Events: []Event{Before}}}, discardLogger(), wg.Go, nil)

	start := time.Now()
	d.Fire(Before, message{Name: "Jane Doe"})
	elapsed := time.Since(start)

	close(release)
	wg.Wait()
	srv.Close()

	if elapsed > 100*time.Millisecond {
		t.Errorf("Fire blocked for %s", elapsed)
	}
}

func TestFireReplacesInvalidUTF8(t *testing.T) {
	rec, srv := newRecorder(t, http.StatusOK)
	var wg sync.WaitGroup
	d := New([]Webhook{{Name: "crm", URL: srv.URL, Method: http.MethodPost, Events: []Event{Success}}}, discardLogger(), wg.Go, nil)

	// "Jürgen" encoded as ISO-8859-1, as sent by a legacy form.
	d.Fire(Success, message{Name: "J\xfcrgen"})
	wg.Wait()

	if len(rec.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(rec.requests))
	}
	if got := rec.requests[0].Body["payload"].(map[string]any)["name"]; got != "J�rgen" {
		t.Errorf("got name %q, want %q", got, "J�rgen")
	}
}

func TestFireDoesNotFollowRedirects(t *testing.T) {
	var (
		mu       sync.Mutex
		redirect bool
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/landed" {
			mu.Lock()
			redirect = true
			mu.Unlock()
			return
		}
		http.Redirect(w, r, "/landed", http.StatusFound)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	var wg sync.WaitGroup
	d := New([]Webhook{{Name: "crm", URL: srv.URL + "/hook", Method: http.MethodPost, Events: []Event{Success}}}, slog.New(slog.NewTextHandler(&buf, nil)), wg.Go, nil)

	d.Fire(Success, message{Name: "Jane Doe"})
	wg.Wait()

	if redirect {
		t.Error("expected the redirect not to be followed")
	}
	if out := buf.String(); !strings.Contains(out, "unexpected status 302") {
		t.Errorf("expected the redirect to be logged as a failure, got %q", out)
	}
}

func TestFireDoesNotLogURLSecrets(t *testing.T) {
	_, failing := newRecorder(t, http.StatusInternalServerError)
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	tests := []struct {
		name string
		host string
	}{
		{name: "non-2xx response", host: strings.TrimPrefix(failing.URL, "http://")},
		{name: "unreachable url", host: strings.TrimPrefix(closed.URL, "http://")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			var wg sync.WaitGroup
			url := "http://user:pass-secret@" + tt.host + "/services/path-secret?token=query-secret"
			d := New([]Webhook{{Name: "crm", URL: url, Method: http.MethodPost, Events: []Event{Error}}}, slog.New(slog.NewTextHandler(&buf, nil)), wg.Go, nil)

			d.Fire(Error, message{Name: "Jane Doe"})
			wg.Wait()

			out := buf.String()
			if !strings.Contains(out, `msg="webhook failed"`) {
				t.Fatalf("expected the failure to be logged, got %q", out)
			}
			for _, secret := range []string{"pass-secret", "path-secret", "query-secret"} {
				if strings.Contains(out, secret) {
					t.Errorf("log contains %q: %s", secret, out)
				}
			}
		})
	}
}
