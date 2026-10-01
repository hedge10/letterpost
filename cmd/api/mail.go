package main

import (
	"net/http"
	"net/mail"

	"hedge10.staticform/internal/mailer"
)

func (app *application) sendMail(w http.ResponseWriter, r *http.Request) {
	fName := r.FormValue("name")
	fSenderAddress := r.FormValue("sender")
	fSubject := r.FormValue("subject")
	fMessageHtml := r.FormValue("html_body")
	fMessagePlain := r.FormValue("plain_body")

	if fSenderAddress == "" {
		http.Error(w, "Missing param sender", http.StatusBadRequest)
		return
	}

	sender := fSenderAddress
	if fName != "" {
		sender = (&mail.Address{Name: fName, Address: fSenderAddress}).String()
	}

	if !mailer.ValidateAddress(sender) {
		http.Error(w, "Invalid email address or name", http.StatusBadRequest)
		return
	}

	var data struct {
		Subject   string
		HtmlBody  string
		PlainBody string
	}
	data.Subject = fSubject
	data.HtmlBody = fMessageHtml
	data.PlainBody = fMessagePlain

	app.background(func() {
		err := app.mailer.Send(sender, "new_mail.tmpl", data)
		if err != nil {
			app.serverErrorResponse(w, r, err)

			return
		}

		app.logger.Info("processed new mail", "sender", sender)
	})

	err := app.writeJSON(w, http.StatusOK, envelope{"email_status": "sent", "email_sender": fSenderAddress}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
