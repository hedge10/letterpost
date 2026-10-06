package mailer

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	nm "net/mail"
	"time"

	tt "text/template"

	"github.com/wneessen/go-mail"
)

//go:embed templates
var templateFS embed.FS

type Mailer struct {
	client   *mail.Client
	from     string
	receiver string
}

// tlsPolicies maps the accepted config values to go-mail's STARTTLS policies.
var tlsPolicies = map[string]mail.TLSPolicy{
	"mandatory":     mail.TLSMandatory,
	"opportunistic": mail.TLSOpportunistic,
	"none":          mail.NoTLS,
}
var defaultTlsPolicy = mail.TLSMandatory

func New(host string, port int, username, password, from, receiver string) (*Mailer, error) {
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

	if receiver == "" || !ValidateAddress(receiver) {
		return nil, errors.New("missing or invalid receiver")
	}

	mailer := &Mailer{
		client:   client,
		from:     from,
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

// Send mails the rendered template to the receiver. The mail is sent from the
// configured from address, as SMTP servers reject senders they don't own, and
// the visitor goes into Reply-To.
func (m *Mailer) Send(name, replyTo, templateFile string, data any) error {
	msg, err := m.buildMsg(name, replyTo, templateFile, data)
	if err != nil {
		return err
	}

	return m.client.DialAndSend(msg)
}

func (m *Mailer) buildMsg(name, replyTo, templateFile string, data any) (*mail.Msg, error) {
	textTmpl, err := tt.New("").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return nil, err
	}

	subject := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return nil, err
	}

	plainBody := new(bytes.Buffer)
	err = textTmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return nil, err
	}

	msg := mail.NewMsg()
	err = msg.FromFormat(name, m.from)
	if err != nil {
		return nil, err
	}
	err = msg.To(m.receiver)
	if err != nil {
		return nil, err
	}
	err = msg.ReplyToFormat(name, replyTo)
	if err != nil {
		return nil, err
	}

	msg.Subject(subject.String())
	msg.SetBodyString(mail.TypeTextPlain, plainBody.String())

	return msg, nil
}

func ValidateAddress(e string) bool {
	_, err := nm.ParseAddress(e)

	return err == nil
}
