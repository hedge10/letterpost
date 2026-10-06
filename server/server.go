package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"codeberg.org/hedge10/staticform/captcha"
	"codeberg.org/hedge10/staticform/webhook"
)

// Config holds the settings of a Server.
type Config struct {
	Port    int    `env:"PORT" envDefault:"4000"`
	Env     string `env:"ENV" envDefault:"prod"`
	Limiter struct {
		RPS     float64 `env:"RPS" envDefault:"2"`
		Burst   int     `env:"BURST" envDefault:"4"`
		Enabled bool    `env:"ENABLED" envDefault:"true"`
	} `envPrefix:"LIMITER_"`
	TrustedOrigins []string `env:"CORS_TRUSTED_ORIGINS" envSeparator:" "`
	HoneypotField  string   `env:"HONEYPOT_FIELD"`
	RedirectURL    string   `env:"REDIRECT_URL"`
}

type Mailer interface {
	Send(name, replyTo, templateFile string, data any) error
}

// Server handles form submissions.
type Server struct {
	config          Config
	logger          *slog.Logger
	mailer          Mailer
	webhooks        *webhook.Dispatcher
	captchaVerifier captcha.Verifier
	wg              sync.WaitGroup
}

func New(cfg Config, logger *slog.Logger, mailer Mailer, captchaVerifier captcha.Verifier, webhooks []webhook.Webhook) (*Server, error) {
	if cfg.RedirectURL != "" {
		if _, ok := originOf(cfg.RedirectURL); !ok {
			return nil, errors.New("redirect URL must be an absolute http(s) URL")
		}
	}

	app := &Server{
		config:          cfg,
		logger:          logger,
		mailer:          mailer,
		captchaVerifier: captchaVerifier,
	}
	app.webhooks = webhook.New(webhooks, logger, app.background, nil)

	return app, nil
}

// Serve listens on the configured port until SIGINT or SIGTERM, then shuts
// down gracefully and waits for background tasks.
func (app *Server) Serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.Port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.Info("stopping server", "addr", srv.Addr, "signal", s.String())
		shutdownError <- srv.Shutdown(context.Background())
	}()

	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.Env)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.Info("waiting for background tasks")
	app.wg.Wait()

	app.logger.Info("shutdown complete")

	return nil
}
