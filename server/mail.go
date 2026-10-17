package server

import (
	"embed"
	"io/fs"
	"net/http"
	"regexp"

	"codeberg.org/hedge10/staticform/mailer"
	"codeberg.org/hedge10/staticform/pipeline"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

//go:embed templates
var templateFiles embed.FS

// MailTemplates holds the templates for the mails staticform sends.
var MailTemplates, _ = fs.Sub(templateFiles, "templates")

type messageInput struct {
	Name string `json:"name"`
	// Email address of the sender
	Sender    string `json:"sender"`
	Subject   string `json:"subject"`
	PlainBody string `json:"plain_body"`
}

var singleLineValidation = regexp.MustCompile(`^[^\r\n]*$`)

func (app *Server) sendMail(w http.ResponseWriter, r *http.Request) {
	message := messageInput{
		Name:      r.PostFormValue("name"),
		Sender:    r.PostFormValue("sender"),
		Subject:   r.PostFormValue("subject"),
		PlainBody: r.PostFormValue("plain_body"),
	}
	err := message.Validate()
	if err != nil {
		app.failedValidationResponse(w, r, err)
		return
	}

	redirect, err := app.redirectTarget(r.PostFormValue("_redirect"))
	if err != nil {
		app.failedValidationResponse(w, r, validation.Errors{"_redirect": err})
		return
	}

	app.pipeline.Deliver(pipeline.Submission{
		Message: mailer.Message{
			To:          []string{app.config.Receiver},
			FromName:    message.Name,
			ReplyTo:     message.Sender,
			ReplyToName: message.Name,
			Template:    "new_mail.tmpl",
			Data:        message,
		},
		Fields: message,
	})

	app.sentResponse(w, r, message.Sender, redirect)
}

func (m messageInput) Validate() error {
	return validation.ValidateStruct(&m,
		validation.Field(&m.Sender, validation.Required, is.Email),
		validation.Field(&m.Subject, validation.RuneLength(0, 100), validation.Match(singleLineValidation)),
		validation.Field(&m.Name, validation.Required, validation.RuneLength(5, 30), validation.Match(singleLineValidation)),
		validation.Field(&m.PlainBody, validation.Required, validation.RuneLength(1, 10000)),
	)
}
