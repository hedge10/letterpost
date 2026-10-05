package main

import (
	"net/http"
	"regexp"

	"hedge10.staticform/internal/webhook"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type messageInput struct {
	Name string `json:"name"`
	// Email address of the sender
	Sender    string `json:"sender"`
	Subject   string `json:"subject"`
	PlainBody string `json:"plain_body"`
}

var singleLineValidation = regexp.MustCompile(`^[^\r\n]*$`)

func (app *application) sendMail(w http.ResponseWriter, r *http.Request) {
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

	app.webhooks.Fire(webhook.Before, message)

	app.background(func() {
		err := app.mailer.Send(message.Name, message.Sender, "new_mail.tmpl", message)
		if err != nil {
			// The response was already sent, so the error can only be logged.
			app.logger.Error("failed to send mail", "sender", message.Sender, "error", err.Error())
			app.webhooks.Fire(webhook.Error, message)
		} else {
			app.logger.Info("processed new mail", "sender", message.Sender)
			app.webhooks.Fire(webhook.Success, message)
		}

		app.webhooks.Fire(webhook.After, message)
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
