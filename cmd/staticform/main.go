package main

import (
	"log/slog"
	"os"

	"codeberg.org/hedge10/staticform/captcha"
	"codeberg.org/hedge10/staticform/mailer"
	"codeberg.org/hedge10/staticform/server"
	"codeberg.org/hedge10/staticform/webhook"

	"github.com/caarlos0/env/v11"
)

const (
	webhooksFile = "webhooks.json"
)

type config struct {
	Server server.Config
	// smtp connection credentials
	Smtp struct {
		Host     string `env:"HOST,required" envDefault:""`
		Port     int    `env:"PORT" envDefault:"25"`
		Username string `env:"USERNAME"`
		Password string `env:"PASSWORD"`
		From     string `env:"FROM,required"`
		Receiver string `env:"RECEIVER,required"`
		Tls      string `env:"TLS" envDefault:"mandatory"`
	} `envPrefix:"SMTP_"`
	// captcha
	Captcha struct {
		Provider string `env:"PROVIDER"`
		Secret   string `env:"SECRET"`
		Sitekey  string `env:"SITEKEY"`
		Enabled  bool   `env:"ENABLED" envDefault:"false"`
	} `envPrefix:"CAPTCHA_"`
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	var cfg config
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: "SF_"}); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	mailer, err := mailer.New(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.From, cfg.Smtp.Receiver)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	err = mailer.SetTlsPolicy(cfg.Smtp.Tls)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	var captchaVerifier captcha.Verifier
	if cfg.Captcha.Enabled {
		captchaVerifier, err = captcha.New(captcha.Provider(cfg.Captcha.Provider), cfg.Captcha.Secret, cfg.Captcha.Sitekey)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	}

	webhooks, err := webhook.Load(webhooksFile)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	logger.Info("loaded webhooks", "count", len(webhooks))

	srv, err := server.New(cfg.Server, logger, mailer, captchaVerifier, webhooks)
	if err != nil {
		logger.Error("cannot create server", "error", err)
		os.Exit(1)
	}

	err = srv.Serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
