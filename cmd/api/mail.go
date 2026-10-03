package main

import (
	"net/http"
	"net/mail"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type messageInput struct {
	Name string `json:"name"`
	// Email address of the sender
	Sender    string `json:"sender"`
	Subject   string `json:"subject"`
	HtmlBody  string `json:"html_body"`
	PlainBody string `json:"plain_body"`
}

var singleLineValidation = regexp.MustCompile(`^[^\r\n]*$`)

func (app *application) sendMail(w http.ResponseWriter, r *http.Request) {
	message := messageInput{
		Name:      r.PostFormValue("name"),
		Sender:    r.PostFormValue("sender"),
		Subject:   r.PostFormValue("subject"),
		HtmlBody:  r.PostFormValue("html_body"),
		PlainBody: r.PostFormValue("plain_body"),
	}
	err := message.Validate()
	if err != nil {
		app.failedValidationResponse(w, r, err)
		return
	}

	app.background(func() {
		sender := (&mail.Address{Name: message.Name, Address: message.Sender}).String()
		err := app.mailer.Send(sender, "new_mail.tmpl", message)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		app.logger.Info("processed new mail", "sender", sender)
	})

	err = app.writeJSON(w, http.StatusOK, envelope{"email_status": "sent", "email_sender": message.Sender}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (m messageInput) Validate() error {
	return validation.ValidateStruct(&m,
		validation.Field(&m.Sender, validation.Required, is.Email),
		validation.Field(&m.Subject, validation.RuneLength(0, 100), validation.Match(singleLineValidation)),
		validation.Field(&m.Name, validation.Required, validation.RuneLength(5, 30), validation.Match(singleLineValidation)),
		validation.Field(&m.PlainBody, validation.Required, validation.RuneLength(1, 10000)),
		validation.Field(&m.HtmlBody, validation.RuneLength(0, 20000)),
	)
}
