# Elasticsearch/OpenSearch TUI Manager Makefile
SHELL := /bin/bash
APP_NAME := es-tui
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
GO ?= go

.PHONY: all build install clean test test-cover test-cover-check lint fmt run start release snapshot \
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

test-cover-check:
	@$(GO) test -v -race -coverprofile=coverage.out ./... && \
		set -euo pipefail; \
		FAILED=0; \
		while IFS= read -r line; do \
			func=$$(echo "$$line" | awk '{print $$2}'); \
			pct=$$(echo "$$line" | awk '{print $$NF}' | tr -d '%'); \
			if [[ "$$func" == "(statements)" ]]; then \
				continue; \
			fi; \
			if (( $$(echo "$$pct < 100.0" | bc -l) )); then \
				location=$$(echo "$$line" | awk '{print $$1}'); \
				echo "FAIL: Function $$func at $$location coverage is $${pct}%, required 100%"; \
				FAILED=1; \
			fi; \
		done < <($(GO) tool cover -func=coverage.out); \
		if [[ $$FAILED -eq 1 ]]; then \
			exit 1; \
		fi; \
		echo "All functions at 100% coverage"

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

help:
	@echo "Available targets:"
	@echo "  build, test, test-cover-check, run, docker-up, docker-seed, docker-down"
