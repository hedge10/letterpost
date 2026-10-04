package main

import (
	"log/slog"
	"os"
	"sync"

	"hedge10.staticform/internal/captcha"
	"hedge10.staticform/internal/mailer"

	"github.com/caarlos0/env/v11"
)

type config struct {
	Port int    `env:"PORT" envDefault:"4000"`
	Env  string `env:"ENV" envDefault:"prod"`
	// rate limiter
	Limiter struct {
		RPS     float64 `env:"RPS" envDefault:"2"`
		Burst   int     `env:"BURST" envDefault:"4"`
		Enabled bool    `env:"ENABLED" envDefault:"true"`
	} `envPrefix:"LIMITER_"`
	// smtp connection credentials
	Smtp struct {
		Host     string `env:"HOST,required" envDefault:""`
		Port     int    `env:"PORT" envDefault:"25"`
		Username string `env:"USERNAME"`
		Password string `env:"PASSWORD"`
		Receiver string `env:"RECEIVER,required"`
		Tls      string `env:"TLS" envDefault:"mandatory"`
	} `envPrefix:"SMTP_"`
	// cors
	Cors struct {
		TrustedOrigins []string `env:"TRUSTED_ORIGINS" envSeparator:" "`
	} `envPrefix:"CORS_"`
	// captcha
	Captcha struct {
		Provider string `env:"PROVIDER"`
		Secret   string `env:"SECRET"`
		Sitekey  string `env:"SITEKEY"`
		Enabled  bool   `env:"ENABLED" envDefault:"false"`
	} `envPrefix:"CAPTCHA_"`
	// honeypot, disabled when empty
	HoneypotField string `env:"HONEYPOT_FIELD"`
	// success page after submission, empty for a JSON response
	RedirectURL string `env:"REDIRECT_URL"`
}

type application struct {
	config          config
	logger          *slog.Logger
	mailer          *mailer.Mailer
	captchaVerifier captcha.Verifier
	wg              sync.WaitGroup
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	var cfg config
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: "SF_"}); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	if _, ok := originOf(cfg.RedirectURL); cfg.RedirectURL != "" && !ok {
		logger.Error("SF_REDIRECT_URL must be an absolute http(s) URL")
		os.Exit(1)
	}

	mailer, err := mailer.New(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.Receiver)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	err = mailer.SetTlsPolicy(cfg.Smtp.Tls)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	app := &application{
		config: cfg,
		logger: logger,
		mailer: mailer,
	}

	if cfg.Captcha.Enabled {
		app.captchaVerifier, err = captcha.New(captcha.Provider(cfg.Captcha.Provider), cfg.Captcha.Secret, cfg.Captcha.Sitekey)
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
