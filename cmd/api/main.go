package main

import (
	"flag"
	"log/slog"
	"os"
	"strings"
	"sync"

	"hedge10.staticform/internal/captcha"
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
	// captcha
	captcha struct {
		provider string
		secret   string
		sitekey  string
		enabled  bool
	}
}

type application struct {
	config  config
	logger  *slog.Logger
	mailer  *mailer.Mailer
	captcha captcha.Verifier
	wg      sync.WaitGroup
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
	flag.StringVar(&cfg.smtp.tls, "smtp-tls", "mandatory", "SMTP STARTTLS policy (mandatory|opportunistic|none)")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.captcha.provider, "captcha-provider", "", "Captcha provider (cloudflare | friendly-captcha)")
	flag.StringVar(&cfg.captcha.secret, "captcha-secret", "", "Captcha secret")
	flag.StringVar(&cfg.captcha.sitekey, "captcha-sitekey", "", "Captcha sitekey (optional)")
	flag.BoolVar(&cfg.captcha.enabled, "captcha-enabled", false, "Enable captcha support")

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

	if cfg.captcha.enabled {
		app.captcha, err = captcha.New(captcha.Provider(cfg.captcha.provider), cfg.captcha.secret, cfg.captcha.sitekey)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
