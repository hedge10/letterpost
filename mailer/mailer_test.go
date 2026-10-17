package mailer

import (
	"testing"
	"testing/fstest"

	"github.com/wneessen/go-mail"
)

type testData struct {
	Subject   string
	PlainBody string
}

var testTemplates = fstest.MapFS{
	"test.tmpl": {Data: []byte(`{{define "subject"}}{{.Subject}}{{end}}{{define "plainBody"}}{{.PlainBody}}{{end}}`)},
}

func newTestMailer(t *testing.T) *Mailer {
	t.Helper()
	m, err := New("smtp.example.com", 25, "", "", "forms@example.com", testTemplates)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNewRequiresValidFrom(t *testing.T) {
	for _, from := range []string{"", "not-an-address"} {
		_, err := New("smtp.example.com", 25, "", "", from, testTemplates)
		if err == nil {
			t.Errorf("expected an error for from %q", from)
		}
	}
}

func TestBuildMsg(t *testing.T) {
	m := newTestMailer(t)

	msg, err := m.buildMsg(Message{
		To:          []string{"owner@example.com"},
		FromName:    "Jane Doe",
		ReplyTo:     "jane.doe@gmail.com",
		ReplyToName: "Jane Doe",
		Template:    "test.tmpl",
		Data:        testData{Subject: "Hello", PlainBody: "Hi"},
	})
	if err != nil {
		t.Fatal(err)
	}

	assertHeader(t, "From", msg.GetFromString(), `"Jane Doe" <forms@example.com>`)
	assertHeader(t, "Reply-To", msg.GetAddrHeaderString(mail.HeaderReplyTo), `"Jane Doe" <jane.doe@gmail.com>`)
	assertHeader(t, "To", msg.GetToString(), "<owner@example.com>")
	assertHeader(t, "Subject", msg.GetGenHeader(mail.HeaderSubject), "Hello")
}

func TestBuildMsgWithoutReplyTo(t *testing.T) {
	m := newTestMailer(t)

	msg, err := m.buildMsg(Message{
		To:       []string{"owner@example.com"},
		FromName: "letterpost",
		Template: "test.tmpl",
		Data:     testData{Subject: "Verify", PlainBody: "Click"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := msg.GetAddrHeaderString(mail.HeaderReplyTo); len(got) != 0 {
		t.Errorf("Reply-To = %q, want none", got)
	}
	assertHeader(t, "From", msg.GetFromString(), `"letterpost" <forms@example.com>`)
}

func TestBuildMsgMultipleRecipients(t *testing.T) {
	m := newTestMailer(t)

	msg, err := m.buildMsg(Message{
		To:       []string{"a@example.com", "b@example.com"},
		Template: "test.tmpl",
		Data:     testData{},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := msg.GetToString()
	if len(got) != 2 || got[0] != "<a@example.com>" || got[1] != "<b@example.com>" {
		t.Errorf("To = %q, want [<a@example.com> <b@example.com>]", got)
	}
}

func TestBuildMsgRequiresValidRecipient(t *testing.T) {
	m := newTestMailer(t)

	for _, to := range [][]string{nil, {"nope"}} {
		_, err := m.buildMsg(Message{To: to, Template: "test.tmpl", Data: testData{}})
		if err == nil {
			t.Errorf("expected an error for To %q", to)
		}
	}
}

func assertHeader(t *testing.T, name string, got []string, want string) {
	t.Helper()
	if len(got) != 1 || got[0] != want {
		t.Errorf("%s = %q, want [%q]", name, got, want)
	}
}
