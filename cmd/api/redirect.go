package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

var errRedirectNotAllowed = errors.New("requires SF_REDIRECT_URL to be set")

func (app *application) redirectTarget(path string) (string, error) {
	if path == "" {
		return app.config.RedirectURL, nil
	}

	origin, ok := originOf(app.config.RedirectURL)
	if !ok {
		return "", errRedirectNotAllowed
	}

	return origin + "/" + strings.TrimLeft(path, "/"), nil
}

func originOf(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}

	return u.Scheme + "://" + u.Host, true
}

func (app *application) sentResponse(w http.ResponseWriter, r *http.Request, sender, target string) {
	if target != "" {
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"email_status": "sent", "email_sender": sender}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
