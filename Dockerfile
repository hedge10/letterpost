FROM golang:1.27.1-alpine3.24 AS builder
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -o staticform ./cmd/api

FROM alpine:3.24
COPY --from=builder /app/staticform /staticform

ENV SF_PORT=4000 \
    SF_ENV=dev \
    SF_SMTP_HOST=sandbox.smtp.mailtrap.io \
    SF_SMTP_PORT=25 \
    SF_SMTP_USERNAME=demo-user \
    SF_SMTP_RECEIVER=jane.doe@example.com \
    SF_SMTP_TLS=mandatory \
    SF_CORS_TRUSTED_ORIGINS="" \
    SF_LIMITER_ENABLED=true \
    SF_LIMITER_RPS=2 \
    SF_LIMITER_BURST=4 \
    SF_CAPTCHA_ENABLED=false \
    SF_CAPTCHA_PROVIDER="" \
    SF_CAPTCHA_SITEKEY=""

ENTRYPOINT ["/bin/sh", "-c", "exec /staticform \
    -port=\"$SF_PORT\" \
    -env=\"$SF_ENV\" \
    -smtp-host=\"$SF_SMTP_HOST\" \
    -smtp-port=\"$SF_SMTP_PORT\" \
    -smtp-username=\"$SF_SMTP_USERNAME\" \
    ${SF_SMTP_PASSWORD:+-smtp-password=\"$SF_SMTP_PASSWORD\"} \
    -smtp-receiver=\"$SF_SMTP_RECEIVER\" \
    -smtp-tls=\"$SF_SMTP_TLS\" \
    -cors-trusted-origins=\"$SF_CORS_TRUSTED_ORIGINS\" \
    -limiter-enabled=\"$SF_LIMITER_ENABLED\" \
    -limiter-rps=\"$SF_LIMITER_RPS\" \
    -limiter-burst=\"$SF_LIMITER_BURST\" \
    -captcha-enabled=\"$SF_CAPTCHA_ENABLED\" \
    -captcha-provider=\"$SF_CAPTCHA_PROVIDER\" \
    ${SF_CAPTCHA_SECRET:+-captcha-secret=\"$SF_CAPTCHA_SECRET\"} \
    -captcha-sitekey=\"$SF_CAPTCHA_SITEKEY\" \
    \"$@\"", "--"]
