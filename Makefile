# Makefile
SHELL := /bin/bash

# ---- Config ----
GO        ?= go
MAIN      ?= ./cmd/api        # change to ./ if your main.go is in repo root
PKG       ?= ./...
ENV_FILE  ?= .env
BIN_DIR   ?= bin
BIN       ?= $(BIN_DIR)/api

COVER_OUT ?= coverage.out
COVER_HTML?= coverage.html
COVER_MIN ?= 80.0             # fail 'cover-check' if below this %

# ---- Default ----
.PHONY: help
help:
	@echo "Usage:"
	@echo "  make run            - run the API (go run $(MAIN))"
	@echo "  make build          - build binary to $(BIN)"
	@echo "  make test           - run tests with race"
	@echo "  make cover          - run tests with coverage (atomic) -> $(COVER_OUT)"
	@echo "  make cover-html     - generate HTML report -> $(COVER_HTML)"
	@echo "  make cover-check    - assert total coverage >= $(COVER_MIN)%"
	@echo "  make fmt vet tidy   - code hygiene"
	@echo "  make deps-up        - docker compose up -d (MySQL + Redis)"
	@echo "  make deps-down      - docker compose down"
	@echo "  make deps-clean     - docker compose down -v (remove volumes)"

# ---- App ----
.PHONY: run
run:
	$(GO) run $(MAIN)

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) $(MAIN)
	@echo "Built $(BIN)"

# ---- Tests & Coverage ----
.PHONY: test
test:
	$(GO) test -race -v $(PKG)

.PHONY: cover
cover:
	$(GO) test -race -covermode=atomic -coverprofile=$(COVER_OUT) $(PKG)
	@$(GO) tool cover -func=$(COVER_OUT) | tail -n1

.PHONY: cover-html
cover-html: cover
	@$(GO) tool cover -html=$(COVER_OUT) -o $(COVER_HTML)
	@echo "Open $(COVER_HTML) in your browser."

.PHONY: cover-check
cover-check: cover
	@cov=$$($(GO) tool cover -func=$(COVER_OUT) | awk '/total:/ {print $$3}' | sed 's/%//'); \
	req=$(COVER_MIN); \
	echo "Total coverage: $$cov% (minimum $$req%)"; \
	awk 'BEGIN {exit !('"$$cov"' >= '"$$req"')}'

# ---- Hygiene ----
.PHONY: fmt
fmt:
	$(GO) fmt $(PKG)

.PHONY: vet
vet:
	$(GO) vet $(PKG)

.PHONY: tidy
tidy:
	$(GO) mod tidy

# ---- Docker deps (MySQL + Redis) ----
.PHONY: deps-up
deps-up:
	docker compose up -d

.PHONY: deps-down
deps-down:
	docker compose down

.PHONY: deps-clean
deps-clean:
	docker compose down -v

# ---- CI convenience ----
.PHONY: ci
ci: fmt vet tidy cover-check
