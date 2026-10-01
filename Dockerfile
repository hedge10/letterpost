FROM golang:1.27.1-alpine3.24 AS builder
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -o staticform ./cmd/api

FROM scratch
COPY --from=builder /app/staticform /staticform

