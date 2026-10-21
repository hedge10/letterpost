package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/tomasen/realip"

	ic "codeberg.org/hedge10/staticform/captcha"
	"codeberg.org/hedge10/staticform/redirect"
)

func (app *Server) captcha(next http.Handler) http.Handler {
	if app.captchaVerifier == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.PostFormValue(app.captchaVerifier.FormField())
		if token == "" {
			app.captchaFailedResponse(w, r)
			return
		}

		err := app.captchaVerifier.Verify(r.Context(), token, realip.FromRequest(r))
		if err != nil {
			if errors.Is(err, ic.ErrInvalidToken) {
				app.captchaFailedResponse(w, r)
				return
			}

			app.serverErrorResponse(w, r, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *Server) honeypot(next http.Handler) http.Handler {
	if app.config.HoneypotField == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.PostFormValue(app.config.HoneypotField) == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Pretend the mail was sent, so bots get no signal that they were caught.
		app.logger.Info("honeypot triggered, mail discarded", "ip", realip.FromRequest(r))
		target, _ := redirect.Target(app.config.RedirectURL, r.PostFormValue("_redirect"))
		app.sentResponse(w, r, r.PostFormValue("sender"), target)
	})
}

func (app *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// Use the built-in recover() function to check if a panic occurred.
			// If a panic did happen, recover() will return the panic value. If
			// a panic didn't happen, it will return nil.
			pv := recover()
			if pv != nil {
				// If there was a panic, set a "Connection: close" header on the
				// response. This acts as a trigger to make Go's HTTP server
				// automatically close the current connection after the response has been
				// sent.
				w.Header().Set("Connection", "close")
				// The value returned by recover() has the type any, so we use
				// fmt.Errorf() with the %v verb to coerce it into an error and
				// call our serverErrorResponse() helper. In turn, this will log the
				// error at the ERROR level and send the client a 500 Internal
				// Server Error response.
				app.serverErrorResponse(w, r, fmt.Errorf("%v", pv))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
