NAME=sorobangraph.attest.so
VERSION=0.0.1

.PHONY: build
## build: Compile the packages.
build:
	@go build -o $(NAME)

.PHONY: run
## run: Build and Run in development mode.
run: build
	@./$(NAME) -e development

.PHONY: run-prod
## run-prod: Build and Run in production mode.
run-prod: build
	@./$(NAME) -e production

.PHONY: clean
## clean: Clean project and previous builds.
clean:
	@rm -f $(NAME)

.PHONY: deps
## deps: Download modules
deps:
	@go mod download

.PHONY: test
## test: Run tests with verbose mode
test:
	@go test -v ./...

# --- New targets ---
.PHONY: run-main
## run-main: Run the main service using `go run` (uses `main.go`).
run-main:
	@go run ./main.go

.PHONY: migrate-up
## migrate-up: Apply database migrations (runs `cmd/migrate up`).
migrate-up:
	@go run ./cmd/migrate up

.PHONY: migrate-status
## migrate-status: Check database connectivity/status (runs `cmd/migrate status`).
migrate-status:
	@go run ./cmd/migrate status

.PHONY: healthcheck
## healthcheck: Run the healthcheck utility (runs `cmd/healthcheck`).
healthcheck:
	@go run ./cmd/healthcheck

.PHONY: build-migrate build-healthcheck
## build-migrate: Build migrate command to `bin/migrate`.
## build-healthcheck: Build healthcheck command to `bin/healthcheck`.
build-migrate:
	@mkdir -p bin
	@go build -o bin/migrate ./cmd/migrate

build-healthcheck:
	@mkdir -p bin
	@go build -o bin/healthcheck ./cmd/healthcheck

.PHONY: help
all: help
# help: show this help message
help: Makefile
	@echo
	@echo " Choose a command to run in "$(APP_NAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
