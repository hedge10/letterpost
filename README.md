# StaticForm - A form service backend

**StaticForm** is a form service backend, that converts form submissions into emails.
It can be used with good old static HTML forms or Javascript-based ones.

## Features

* Handle form data (`multipart/form-data`)
* Input validation with sane defaults
* IP-based rate limiting
* Spam Protection
  * Honeypot field supported
  * Captcha integration
    * [Cloudflare Turnstile](https://www.cloudflare.com/products/turnstile/)
    * [Friendly Captcha](https://friendlycaptcha.com/)
* Redirect to a success page after submission
* Webhooks on submission lifecycle events
* Support TLS- and non-encrypted SMTP connections
* Healthcheck endpoint for deployments

## Quickstart

Start the container:

```sh
docker run -d -p 4000:4000 \
  -e SF_SMTP_HOST=smtp.example.com \
  -e SF_SMTP_USERNAME=user \
  -e SF_SMTP_PASSWORD=secret \
  -e SF_SMTP_FROM=forms@example.com \
  -e SF_SMTP_RECEIVER=you@example.com \
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
| `SF_SMTP_FROM`            | _none_ (**required**)      | Address the mails are sent from, must be owned by the SMTP account. Visitor goes in Reply-To  |
| `SF_SMTP_RECEIVER`        | _none_ (**required**)      | Email address that receives the form submissions                                              |
| `SF_SMTP_TLS`             | `mandatory`                | TLS policy for the SMTP connection: `mandatory`, `opportunistic` or `none`                    |
| `SF_CORS_TRUSTED_ORIGINS` | _none_                     | Space-separated list of origins allowed to send cross-origin requests                         |
| `SF_CAPTCHA_ENABLED`      | `false`                    | Enable captcha verification                                                                   |
| `SF_CAPTCHA_PROVIDER`     | _none_                     | Captcha provider: `cloudflare` or `friendly-captcha`                                          |
| `SF_CAPTCHA_SECRET`       | _none_                     | Secret / API key of the captcha provider, required when captcha is enabled                    |
| `SF_CAPTCHA_SITEKEY`      | _none_                     | Sitekey, only used by Friendly Captcha (optional)                                             |
| `SF_HONEYPOT_FIELD`       | _none_                     | Name of the honeypot form field. If set and the field is filled, the mail is silently dropped |
| `SF_REDIRECT_URL`         | _none_                     | Success page visitors are redirected to after submitting. If unset, a JSON response is sent   |

#### Redirect after submission

By default `/v1/send` answers with JSON, which suits Javascript-based forms. For plain HTML forms,
set `SF_REDIRECT_URL` and visitors are redirected (`303 See Other`) to that page after submitting.

A form can choose a different success page on the same site with a `_redirect` field. Its value is
used as a path on the origin of `SF_REDIRECT_URL`:

```html
<input type="hidden" name="_redirect" value="/thank-you">
```

With `SF_REDIRECT_URL=https://example.com` this redirects to `https://example.com/thank-you`.
A leading `/` is optional. Sending `_redirect` while `SF_REDIRECT_URL` is unset is rejected with `422`.

Note: when `SF_REDIRECT_URL` is set, every submission is redirected, including those from Javascript clients.

#### Webhooks

StaticForm can notify other services about submissions. Webhooks are defined in a `webhooks.json`
file in the working directory.

```json
[
  {
    "name": "crm-sync",
    "url": "https://crm.example.com/hook",
    "method": "POST",
    "events": ["success", "error"]
  }
]
```

| Field    | Description                                                    |
|----------|----------------------------------------------------------------|
| `name`   | Name of the webhook, used in logs and the payload              |
| `url`    | Absolute `http` or `https` URL that is called                  |
| `method` | `POST`, `PUT` or `PATCH`                                       |
| `events` | One or more of `before`, `after`, `error`, `success`           |

| Event     | Called                                           |
|-----------|--------------------------------------------------|
| `before`  | Before the mail is sent                          |
| `success` | After the mail was sent                          |
| `error`   | After sending the mail failed                    |
| `after`   | After `success` or `error`, regardless of result |

Each request has `Content-Type: application/json` and contains the message fields in `payload`, plus `metadata`:

```json
{
  "payload": {
    "name": "Jane Doe",
    "sender": "jane.doe@example.com",
    "subject": "Hello",
    "plain_body": "Some demo message"
  },
  "metadata": {
    "event_name": "success",
    "webhook_name": "crm-sync"
  }
}
```

* Webhooks are fire-and-forget: they are called in the background and never read the response.
* Redirects are not followed, so use the final URL.
* Failures are only logged, with the webhook name and host but not the full URL.
* Events are only fired for submissions that passed validation and the honeypot and captcha checks.

Mount the file at `/webhooks.json` in the Docker image.

```sh
docker run -d -p 4000:4000 -v ./webhooks.json:/webhooks.json:ro ghcr.io/hedge10/staticform:latest
```
