package main

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func validMessage() messageInput {
	return messageInput{
		Name:      "Jane Doe",
		Sender:    "jane.doe@gmail.com",
		Subject:   "Hello",
		PlainBody: "Some demo message",
	}
}

func TestMessageInputValidate(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(m *messageInput)
		wantField string
	}{
		{name: "valid", modify: func(m *messageInput) {}},
		{name: "subject and html body are optional", modify: func(m *messageInput) { m.Subject = ""; m.HtmlBody = "" }},
		{name: "missing sender", modify: func(m *messageInput) { m.Sender = "" }, wantField: "sender"},
		{name: "sender is not an email address", modify: func(m *messageInput) { m.Sender = "not-an-address" }, wantField: "sender"},
		{name: "sender with name is rejected", modify: func(m *messageInput) { m.Sender = `"Jane" <jane.doe@gmail.com>` }, wantField: "sender"},
		{name: "missing name", modify: func(m *messageInput) { m.Name = "" }, wantField: "name"},
		{name: "name too short", modify: func(m *messageInput) { m.Name = "Jane" }, wantField: "name"},
		{name: "name at min length", modify: func(m *messageInput) { m.Name = "Janet" }},
		{name: "name at max length", modify: func(m *messageInput) { m.Name = strings.Repeat("a", 30) }},
		{name: "name too long", modify: func(m *messageInput) { m.Name = strings.Repeat("a", 31) }, wantField: "name"},
		{name: "name length counts characters, not bytes", modify: func(m *messageInput) { m.Name = strings.Repeat("ä", 30) }},
		{name: "missing plain body", modify: func(m *messageInput) { m.PlainBody = "" }, wantField: "plain_body"},
		{name: "name with line feed", modify: func(m *messageInput) { m.Name = "Jane\nDoe" }, wantField: "name"},
		{name: "name with carriage return", modify: func(m *messageInput) { m.Name = "Jane\rDoe" }, wantField: "name"},
		{name: "name with trailing CRLF", modify: func(m *messageInput) { m.Name = "Jane Doe\r\n" }, wantField: "name"},
		{name: "subject at max length", modify: func(m *messageInput) { m.Subject = strings.Repeat("a", 100) }},
		{name: "subject too long", modify: func(m *messageInput) { m.Subject = strings.Repeat("a", 101) }, wantField: "subject"},
		{name: "subject length counts characters, not bytes", modify: func(m *messageInput) { m.Subject = strings.Repeat("ä", 100) }},
		{name: "subject with line feed", modify: func(m *messageInput) { m.Subject = "Hello\nBcc: victim@example.com" }, wantField: "subject"},
		{name: "subject with carriage return", modify: func(m *messageInput) { m.Subject = "Hello\rWorld" }, wantField: "subject"},
		{name: "subject with trailing CRLF", modify: func(m *messageInput) { m.Subject = "Hello\r\n" }, wantField: "subject"},
		{name: "plain body may contain line breaks", modify: func(m *messageInput) { m.PlainBody = "Line one\r\nLine two\nLine three" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := validMessage()
			tt.modify(&m)

			err := m.Validate()

			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}

			var errs validation.Errors
			if !errors.As(err, &errs) {
				t.Fatalf("expected validation.Errors, got %v", err)
			}
			if _, ok := errs[tt.wantField]; !ok {
				t.Errorf("expected an error for %q, got %v", tt.wantField, errs)
			}
		})
	}
}

func TestSendMailRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		form url.Values
	}{
		{name: "empty form", form: url.Values{}},
		{name: "missing sender", form: url.Values{"name": {"Jane Doe"}, "plain_body": {"Hi"}}},
		{name: "invalid sender", form: url.Values{"name": {"Jane Doe"}, "sender": {"nope"}, "plain_body": {"Hi"}}},
		{name: "missing plain body", form: url.Values{"name": {"Jane Doe"}, "sender": {"jane.doe@gmail.com"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &application{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

			r := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(tt.form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			app.sendMail(w, r)
			app.wg.Wait()

			if w.Code < 400 || w.Code >= 500 {
				t.Errorf("expected a 4xx status, got %d", w.Code)
			}
			if body := w.Body.String(); strings.Contains(body, "email_status") {
				t.Errorf("invalid input must not be reported as sent, body: %s", body)
			}
		})
	}
}
