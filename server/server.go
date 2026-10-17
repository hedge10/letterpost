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
	"codeberg.org/hedge10/staticform/mailer"
	"codeberg.org/hedge10/staticform/pipeline"
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
	// Receiver gets the form submissions.
	Receiver      string `env:"SMTP_RECEIVER,required"`
	HoneypotField string `env:"HONEYPOT_FIELD"`
	RedirectURL   string `env:"REDIRECT_URL"`
}

type Mailer interface {
	Send(msg mailer.Message) error
}

// Server handles form submissions.
type Server struct {
	config          Config
	logger          *slog.Logger
	pipeline        *pipeline.Pipeline
	captchaVerifier captcha.Verifier
	wg              sync.WaitGroup
}

func New(cfg Config, logger *slog.Logger, m Mailer, captchaVerifier captcha.Verifier, webhooks []webhook.Webhook) (*Server, error) {
	if !mailer.ValidateAddress(cfg.Receiver) {
		return nil, errors.New("missing or invalid receiver")
	}

	if cfg.RedirectURL != "" {
		if _, ok := originOf(cfg.RedirectURL); !ok {
			return nil, errors.New("redirect URL must be an absolute http(s) URL")
		}
	}

	app := &Server{
		config:          cfg,
		logger:          logger,
		captchaVerifier: captchaVerifier,
	}
	app.pipeline = app.newPipeline(m, webhook.New(webhooks, logger, app.background, nil))

	return app, nil
}

// newPipeline sends with m, logs every delivery and fires the webhooks.
func (app *Server) newPipeline(m Mailer, d *webhook.Dispatcher) *pipeline.Pipeline {
	p := pipeline.New(m.Send, app.logger, app.background)
	p.On(app.logDelivery, pipeline.Success, pipeline.Error)
	p.On(d.Hook(), pipeline.Events()...)
	return p
}

// logDelivery logs the result of sending a mail.
func (app *Server) logDelivery(_ pipeline.Event, s pipeline.Submission, err error) {
	if err != nil {
		app.logger.Error("failed to send mail", "sender", s.Message.ReplyTo, "error", err.Error())
		return
	}
	app.logger.Info("processed new mail", "sender", s.Message.ReplyTo)
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
