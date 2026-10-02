# StaticForm - A form service backend

**StaticForm** is a form service backend, that converts form submissions into emails.
It can be used with good old static HTML forms or Javascript-based ones.

## Features

* Handle form data (`multipart/form-data`)
* IP-based rate limiting
* Captcha integration:
  * [Cloudflare Turnstile](https://www.cloudflare.com/products/turnstile/)
  * [Friendly Captcha](https://friendlycaptcha.com/)
* Support TLS- and non-encrypted SMTP connections
* Healthcheck endpoint for deployments

