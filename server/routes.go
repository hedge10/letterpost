package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

	"codeberg.org/hedge10/staticform/middleware"
)

func (app *Server) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/health", app.healthcheckHandler)
	router.Handler(http.MethodPost, "/v1/send", app.honeypot(app.captcha(http.HandlerFunc(app.sendMail))))

	limited := middleware.RateLimit(app.config.Limiter, router, app.rateLimitExceededResponse)
	return app.recoverPanic(middleware.CORS(app.config.TrustedOrigins, limited))
}
