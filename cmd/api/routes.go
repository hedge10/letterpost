package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.Handler(http.MethodPost, "/v1/send", app.honeypot(app.captcha(http.HandlerFunc(app.sendMail))))

	return app.recoverPanic(app.enableCORS(app.rateLimit(router)))
}
