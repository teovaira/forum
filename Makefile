# Makefile — build/run/test shortcuts for the forum project.
#
# `make docker` is what satisfies the audit's "does the project present a
# script to build the images and containers" check, so it must genuinely
# build and run the container rather than only document the commands.
#
# Run `make` or `make help` for the full target list.

# Configuration — override any of these on the command line, e.g.
#   make run PORT=9090
#   make docker DOCKER_TAG=forum:dev
BINARY      ?= server
CMD_PATH    ?= ./cmd/server
PORT        ?= 8080
DB_PATH     ?= ./forum.db
DOCKER_IMAGE?= forum
DOCKER_TAG  ?= latest
DOCKER_NAME ?= forum
COVERAGE    ?= coverage.txt
COVERAGE_HTML ?= coverage.html

# Every target is a command, not a file, so none of them should be skipped
# because a same-named file happens to exist.
.PHONY: help all build run dev test test-race test-cover cover-html \
        fmt fmt-check vet lint check tidy clean \
        docker docker-build docker-run docker-stop docker-clean

# `make` with no arguments prints help rather than silently building.
.DEFAULT_GOAL := help

## help: list every available target
help:
	@echo "Forum — available targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  make /'
	@echo ""
	@echo "Configuration (override on the command line):"
	@echo "  BINARY=$(BINARY)  PORT=$(PORT)  DB_PATH=$(DB_PATH)"
	@echo "  DOCKER_IMAGE=$(DOCKER_IMAGE):$(DOCKER_TAG)  DOCKER_NAME=$(DOCKER_NAME)"

## all: run the full quality gate, then build the binary
all: check build

## build: compile the server binary (see BINARY below)
build:
	go build -o $(BINARY) $(CMD_PATH)

## run: start the server (templates and static assets resolve from the repo root)
run:
	PORT=$(PORT) DB_PATH=$(DB_PATH) go run $(CMD_PATH)

## dev: run with SECURE_COOKIES off, stated explicitly for local HTTP
dev:
	PORT=$(PORT) DB_PATH=$(DB_PATH) SECURE_COOKIES=false go run $(CMD_PATH)

## test: run the full test suite
test:
	go test ./...

## test-race: run the suite with the race detector
test-race:
	go test ./... -race

## test-cover: run the suite with coverage and race detection
test-cover:
	go test ./... -v -cover -race

## cover-html: write a coverage profile and open it as HTML
cover-html:
	go test ./... -coverprofile=$(COVERAGE)
	go tool cover -html=$(COVERAGE) -o $(COVERAGE_HTML)
	@echo "coverage report written to $(COVERAGE_HTML)"

## fmt: format every Go file in place
fmt:
	go fmt ./...

## fmt-check: fail if any file is unformatted (for CI; does not modify files)
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "these files are not gofmt-clean:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	@echo "gofmt: clean"

## vet: run the static analyzer
vet:
	go vet ./...

## lint: formatting check plus static analysis
lint: fmt-check vet

## check: the full quality gate — formatting, vet, and the race-enabled suite
check: fmt-check vet test-race

## tidy: prune and verify module dependencies
tidy:
	go mod tidy
	go mod verify

## clean: remove build artifacts, coverage output, and the local database
clean:
	rm -f $(BINARY) $(COVERAGE) $(COVERAGE_HTML)
	rm -f $(DB_PATH)
	go clean

## docker: build the image and run the container (the audit's build script)
docker: docker-build docker-run

## docker-build: build the container image
docker-build:
	@if [ ! -f Dockerfile ]; then \
		echo "Dockerfile not found at the repository root."; \
		exit 1; \
	fi
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

## docker-run: start the container, replacing any previous one of the same name
docker-run:
	-docker rm -f $(DOCKER_NAME) 2>/dev/null
	docker run -d --name $(DOCKER_NAME) -p $(PORT):8080 $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "container $(DOCKER_NAME) listening on http://localhost:$(PORT)"

## docker-stop: stop and remove the running container
docker-stop:
	-docker rm -f $(DOCKER_NAME)

## docker-clean: remove the container and its image
docker-clean: docker-stop
	-docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG)
