# Elasticsearch/OpenSearch TUI Manager Makefile
SHELL := /bin/bash
APP_NAME := es-tui
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
GO ?= go

.PHONY: all build install clean test test-cover test-cover-check lint fmt run start release snapshot demo \
	docker-up docker-down docker-seed \
	docker-up-es docker-up-os docker-down-es docker-down-os \
	docker-seed-es docker-seed-os

all: build

build:
	$(GO) build $(LDFLAGS) -o bin/$(APP_NAME) ./

install:
	$(GO) install $(LDFLAGS) ./

clean:
	rm -rf bin/
	rm -rf dist/

test:
	$(GO) test -v -race ./...

test-cover:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Coverage floor for the library (examples excluded). Critical packages must stay 100%.
MIN_COVERAGE ?= 80

test-cover-check:
	@set -euo pipefail; \
		pkgs=$$($(GO) list ./... | grep -v '/examples/'); \
		$(GO) test -v -race -coverprofile=coverage.out $$pkgs; \
		$(GO) test -v ./examples/...; \
		total=$$($(GO) tool cover -func=coverage.out | awk '/total:/ {print $$NF}' | tr -d '%'); \
		echo "Total coverage: $${total}% (minimum $(MIN_COVERAGE)%)"; \
		if (( $$(echo "$$total < $(MIN_COVERAGE)" | bc -l) )); then \
			echo "FAIL: coverage $${total}% is below $(MIN_COVERAGE)%"; \
			exit 1; \
		fi; \
		for dir in internal/cmd internal/types internal/service; do \
			$(GO) test -coverprofile=/tmp/es-tui-pkg.out ./$$dir >/dev/null; \
			pt=$$($(GO) tool cover -func=/tmp/es-tui-pkg.out | awk '/total:/ {print $$NF}' | tr -d '%'); \
			echo "$$dir coverage: $${pt}%"; \
			if (( $$(echo "$$pt < 100.0" | bc -l) )); then \
				echo "FAIL: $$dir must remain at 100% coverage (got $${pt}%)"; \
				$(GO) tool cover -func=/tmp/es-tui-pkg.out | awk '$$NF+0 < 100 {print}'; \
				exit 1; \
			fi; \
		done; \
		echo "Coverage OK"

lint:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

run: build
	./bin/$(APP_NAME)

start: run

build-all:
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 ./
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64 ./
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 ./
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/$(APP_NAME)-linux-arm64 ./
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe ./

release:
	goreleaser release --clean

snapshot:
	goreleaser release --snapshot --clean

dev-deps:
	go install github.com/goreleaser/goreleaser/v2@v2.13.1

## --- Docker Examples ---

docker-up: docker-up-es docker-up-os

docker-down: docker-down-es docker-down-os

docker-seed: docker-seed-es docker-seed-os

docker-up-es:
	docker compose -f examples/elasticsearch/docker-compose.yml up -d

docker-down-es:
	docker compose -f examples/elasticsearch/docker-compose.yml down

docker-seed-es:
	$(GO) run ./examples/seed -addr http://localhost:9200 -flush

docker-up-os:
	docker compose -f examples/opensearch/docker-compose.yml up -d

docker-down-os:
	docker compose -f examples/opensearch/docker-compose.yml down

docker-seed-os:
	$(GO) run ./examples/seed -addr http://localhost:9201 -flush

## --- Demo ---

## Render README demo GIF/PNG (VHS settings match redis-tui: Dracula, 1920x1080).
## Built without $(LDFLAGS) so main.version stays "dev" (no update banner).
## Unset NO_COLOR so lipgloss/bubbletea emit colors (agents often set NO_COLOR=1).
demo: docker-up docker-seed
	$(GO) build -o bin/$(APP_NAME) ./
	env -u NO_COLOR COLORTERM=truecolor TERM=xterm-256color CLICOLOR_FORCE=1 vhs docs/demo.tape
	@python3 scripts/pick_demo_frames.py

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Build & Dev:"
	@echo "    build, install, clean, test, test-cover, test-cover-check"
	@echo "    lint, fmt, run, start, build-all, release, snapshot, dev-deps"
	@echo ""
	@echo "  Docker Examples:"
	@echo "    docker-up / docker-down / docker-seed"
	@echo "    docker-up-es / docker-seed-es / docker-down-es"
	@echo "    docker-up-os / docker-seed-os / docker-down-os"
	@echo ""
	@echo "  Demo:"
	@echo "    demo        - Render README demo GIF (requires vhs)"
	@echo ""
	@echo "    help        - Show this help"
