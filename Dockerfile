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
    SF_SMTP_TLS=opportunistic \
    SF_CORS_TRUSTED_ORIGINS=""

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
    \"$@\"", "--"]
