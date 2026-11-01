package server

import (
	"fmt"
	"net/http"

	"codeberg.org/hedge10/staticform/redirect"
)

func (app *Server) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)
	app.logger.Error(err.Error(), "method", method, "uri", uri)
}

func (app *Server) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}
	// Write the response using the writeJSON() helper. If this happens to return an
	// error then log it, and fall back to sending the client an empty response with a
	// 500 Internal Server Error status code.
	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// formErrorResponse answers a failed form submission. With SF_REDIRECT_URL set
// the visitor is redirected with error=code in the query, otherwise the error
// is sent as JSON.
func (app *Server) formErrorResponse(w http.ResponseWriter, r *http.Request, status int, code string, message any) {
	target, ok := redirect.ErrorTarget(app.config.RedirectURL, r.PostFormValue("_redirect_error"), code)
	if target != "" {
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	if !ok && app.config.RedirectURL == "" && r.PostFormValue("_redirect_error") != "" {
		app.logger.Warn("_redirect_error was sent but SF_REDIRECT_URL is not set")
	}

	app.errorResponse(w, r, status, message)
}

func (app *Server) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	message :=
		"the server encountered a problem and could not process your request"
	app.formErrorResponse(w, r, http.StatusInternalServerError, "server_error", message)
}

func (app *Server) failedValidationResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.formErrorResponse(w, r, http.StatusUnprocessableEntity, "invalid", err)
}

func (app *Server) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	app.formErrorResponse(w, r, http.StatusTooManyRequests, "rate_limited", message)
}

func (app *Server) captchaMissingResponse(w http.ResponseWriter, r *http.Request) {
	message := "captcha verification failed"
	app.formErrorResponse(w, r, http.StatusForbidden, "captcha_missing", message)
}

func (app *Server) captchaFailedResponse(w http.ResponseWriter, r *http.Request) {
	message := "captcha verification failed"
	app.formErrorResponse(w, r, http.StatusForbidden, "captcha_failed", message)
}

func (app *Server) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

func (app *Server) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}
