## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: build the Docker container with the current source
.PHONY: build
build:
	docker compose -f compose.dev.yml up -d --build

## run: run the cmd/api application
.PHONY: run
run:
	go run ./cmd/api -smtp-host=localhost -smtp-port=1025 -smtp-tls=none

## dispatch-msg: send a test message to the local API
.PHONY: dispatch-msg
dispatch-msg:
	curl -X POST \
		-F name='John Doe' \
		-F sender=john.doe@example.com \
		-F subject='A new message from Static Form' \
		-F plain_body='Demo Demo Demo. Yippie!' \
		http://localhost:4000/v1/send


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #
## tidy: tidy module dependencies, and format and modernize all .go files
.PHONY: tidy
tidy:
	go mod tidy
	go fix ./...
	go fmt ./...

## audit: run quality control checks
.PHONY: audit
audit:
	go mod tidy -diff
	go mod verify
	go vet ./...
	go tool staticcheck ./...
	go test -race -vet=off ./...
