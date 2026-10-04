package mailer

import (
	"testing"

	"github.com/wneessen/go-mail"
)

type testData struct {
	Subject   string
	PlainBody string
}

func TestNewRequiresValidFrom(t *testing.T) {
	for _, from := range []string{"", "not-an-address"} {
		_, err := New("smtp.example.com", 25, "", "", from, "owner@example.com")
		if err == nil {
			t.Errorf("expected an error for from %q", from)
		}
	}
}

func TestBuildMsg(t *testing.T) {
	m, err := New("smtp.example.com", 25, "", "", "forms@example.com", "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}

	msg, err := m.buildMsg("Jane Doe", "jane.doe@gmail.com", "new_mail.tmpl", testData{Subject: "Hello", PlainBody: "Hi"})
	if err != nil {
		t.Fatal(err)
	}

	assertHeader(t, "From", msg.GetFromString(), `"Jane Doe" <forms@example.com>`)
	assertHeader(t, "Reply-To", msg.GetAddrHeaderString(mail.HeaderReplyTo), `"Jane Doe" <jane.doe@gmail.com>`)
	assertHeader(t, "To", msg.GetToString(), "<owner@example.com>")
}

func assertHeader(t *testing.T, name string, got []string, want string) {
	t.Helper()
	if len(got) != 1 || got[0] != want {
		t.Errorf("%s = %q, want [%q]", name, got, want)
	}
}
