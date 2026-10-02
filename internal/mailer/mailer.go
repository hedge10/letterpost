package mailer

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	nm "net/mail"
	"time"

	ht "html/template"
	tt "text/template"

	"github.com/wneessen/go-mail"
)

//go:embed templates
var templateFS embed.FS

type Mailer struct {
	client   *mail.Client
	receiver string
}

// tlsPolicies maps the accepted config values to go-mail's STARTTLS policies.
var tlsPolicies = map[string]mail.TLSPolicy{
	"mandatory":     mail.TLSMandatory,
	"opportunistic": mail.TLSOpportunistic,
	"none":          mail.NoTLS,
}
var defaultTlsPolicy = mail.TLSMandatory

func New(host string, port int, username, password, receiver string) (*Mailer, error) {
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

	if receiver == "" || !ValidateAddress(receiver) {
		return nil, errors.New("missing or invalid receiver")
	}

	mailer := &Mailer{
		client:   client,
		receiver: receiver,
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

func (m *Mailer) Send(sender string, templateFile string, data any) error {
	textTmpl, err := tt.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlTmpl, err := ht.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = htmlTmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	msg := mail.NewMsg()
	err = msg.From(sender)
	if err != nil {
		return err
	}
	err = msg.To(m.receiver)
	if err != nil {
		return err
	}
	err = msg.ReplyTo(sender)
	if err != nil {
		return err
	}

	msg.Subject(subject.String())
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody.String())

	return m.client.DialAndSend(msg)
}

func ValidateAddress(e string) bool {
	_, err := nm.ParseAddress(e)

	return err == nil
}
