# StaticForm - A form service backend

**StaticForm** is a form service backend, that converts form submissions into emails.
It can be used with good old static HTML forms or Javascript-based ones.

## Features

* Handle form data (`multipart/form-data`)
* Input validation with sane defaults
* IP-based rate limiting
* Spam Protection
  * Honeypot field for spam protection
  * Captcha integration:
    * [Cloudflare Turnstile](https://www.cloudflare.com/products/turnstile/)
    * [Friendly Captcha](https://friendlycaptcha.com/)
* Support TLS- and non-encrypted SMTP connections
* Healthcheck endpoint for deployments

## Quickstart

Start the container:

```sh
docker run -d -p 4000:4000 \
  -e SF_SMTP_HOST=smtp.example.com \
  -e SF_SMTP_USERNAME=user \
  -e SF_SMTP_PASSWORD=secret \
  -e SF_SMTP_RECEIVER=forms@example.com \
  ghcr.io/hedge10/staticform:latest
```

Check that it is up:

```sh
curl http://localhost:4000/v1/healthcheck
```

Forms can now be submitted via `POST` to `http://localhost:4000/v1/send`.

### Configuration

The configuration can be done entirely via environment variables

| Variable                  | Default                    | Description                                                                                   |
|---------------------------|----------------------------|-----------------------------------------------------------------------------------------------|
| `SF_PORT`                 | `4000`                     | Port the HTTP server listens on                                                               |
| `SF_ENV`                  | `prod`                     | Environment name, shown in the healthcheck and logs                                           |
| `SF_LIMITER_ENABLED`      | `true`                     | Enable IP-based rate limiting                                                                 |
| `SF_LIMITER_RPS`          | `2`                        | Allowed requests per second per IP                                                            |
| `SF_LIMITER_BURST`        | `4`                        | Maximum burst of requests per IP                                                              |
| `SF_SMTP_HOST`            | _none_ (**required**)      | SMTP server host                                                                              |
| `SF_SMTP_PORT`            | `25`                       | SMTP server port                                                                              |
| `SF_SMTP_USERNAME`        | _none_                     | SMTP username                                                                                 |
| `SF_SMTP_PASSWORD`        | _none_                     | SMTP password                                                                                 |
| `SF_SMTP_RECEIVER`        | _none_ (**required**)      | Email address that receives the form submissions                                              |
| `SF_SMTP_TLS`             | `mandatory`                | TLS policy for the SMTP connection: `mandatory`, `opportunistic` or `none`                    |
| `SF_CORS_TRUSTED_ORIGINS` | _none_                     | Space-separated list of origins allowed to send cross-origin requests                         |
| `SF_CAPTCHA_ENABLED`      | `false`                    | Enable captcha verification                                                                   |
| `SF_CAPTCHA_PROVIDER`     | _none_                     | Captcha provider: `cloudflare` or `friendly-captcha`                                          |
| `SF_CAPTCHA_SECRET`       | _none_                     | Secret / API key of the captcha provider, required when captcha is enabled                    |
| `SF_CAPTCHA_SITEKEY`      | _none_                     | Sitekey, only used by Friendly Captcha (optional)                                             |
| `SF_HONEYPOT_FIELD`       | _none_                     | Name of the honeypot form field. If set and the field is filled, the mail is silently dropped |
