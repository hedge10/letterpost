# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine3.24 AS builder
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy code and build
COPY . .
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o staticform ./cmd/staticform

FROM alpine:3.24
RUN addgroup -S staticform && adduser -S -G staticform staticform
COPY --from=builder /app/staticform /staticform

EXPOSE 4000
USER staticform:staticform
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -qO- "http://localhost:${SF_PORT:-4000}/health" || exit 1
ENTRYPOINT ["/staticform"]
