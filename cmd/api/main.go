package main

import (
	"flag"
	"log/slog"
	"os"
	"strings"
	"sync"

	"hedge10.staticform/internal/mailer"
)

type config struct {
	port int
	env  string
	// rate limiter
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	// smtp connection credentials
	smtp struct {
		host     string
		port     int
		username string
		password string
		receiver string
		tls      string
	}
	// cors
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config config
	logger *slog.Logger
	mailer *mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "dev", "Environment (dev|prod)")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "demo-user", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "s3cret123", "SMTP password")
	flag.StringVar(&cfg.smtp.receiver, "smtp-receiver", "jane.doe@example.com", "SMTP receiver")
	flag.StringVar(&cfg.smtp.tls, "smtp-tls", "opportunistic", "SMTP STARTTLS policy (mandatory|opportunistic|none)")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mailer, err := mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.receiver)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	err = mailer.SetTlsPolicy(cfg.smtp.tls)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	app := &application{
		config: cfg,
		logger: logger,
		mailer: mailer,
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
