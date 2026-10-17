package mailer

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	nm "net/mail"
	"time"

	tt "text/template"

	"github.com/wneessen/go-mail"
)

type Mailer struct {
	client    *mail.Client
	from      string
	templates fs.FS
}

// Message represents a single mail
type Message struct {
	To          []string
	FromName    string
	ReplyTo     string
	ReplyToName string
	Template    string
	Data        any
}

// tlsPolicies maps the accepted config values to go-mail's STARTTLS policies.
var tlsPolicies = map[string]mail.TLSPolicy{
	"mandatory":     mail.TLSMandatory,
	"opportunistic": mail.TLSOpportunistic,
	"none":          mail.NoTLS,
}
var defaultTlsPolicy = mail.TLSMandatory

// New creates a Mailer that renders mail templates from the given file system.
func New(host string, port int, username, password, from string, templates fs.FS) (*Mailer, error) {
	client, err := mail.NewClient(
		host,
		mail.WithTLSPolicy(defaultTlsPolicy),
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithPort(port),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	if from == "" || !ValidateAddress(from) {
		return nil, errors.New("missing or invalid from address")
	}

	mailer := &Mailer{
		client:    client,
		from:      from,
		templates: templates,
	}

	return mailer, nil
}

func (m *Mailer) SetTlsPolicy(p string) error {
	policy, exists := tlsPolicies[p]
	if !exists {
		return fmt.Errorf("invalid TLS policy %q (mandatory|opportunistic|none)", p)
	}

	m.client.SetTLSPolicy(policy)
	if policy != mail.TLSMandatory {
		m.client.SetSMTPAuth(mail.SMTPAuthLoginNoEnc)
	}

	return nil
}

func (m *Mailer) Send(msg Message) error {
	mm, err := m.buildMsg(msg)
	if err != nil {
		return err
	}

	return m.client.DialAndSend(mm)
}

func (m *Mailer) buildMsg(msg Message) (*mail.Msg, error) {
	if len(msg.To) == 0 {
		return nil, errors.New("missing receiver")
	}

	textTmpl, err := tt.New("").ParseFS(m.templates, msg.Template)
	if err != nil {
		return nil, err
	}

	subject := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(subject, "subject", msg.Data)
	if err != nil {
		return nil, err
	}

	plainBody := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(plainBody, "plainBody", msg.Data)
	if err != nil {
		return nil, err
	}

	mm := mail.NewMsg()
	err = mm.FromFormat(msg.FromName, m.from)
	if err != nil {
		return nil, err
	}
	err = mm.To(msg.To...)
	if err != nil {
		return nil, err
	}
	if msg.ReplyTo != "" {
		err = mm.ReplyToFormat(msg.ReplyToName, msg.ReplyTo)
		if err != nil {
			return nil, err
		}
	}

	mm.Subject(subject.String())
	mm.SetBodyString(mail.TypeTextPlain, plainBody.String())

	return mm, nil
}

func ValidateAddress(e string) bool {
	_, err := nm.ParseAddress(e)

	return err == nil
}
