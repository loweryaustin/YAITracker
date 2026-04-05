.PHONY: help confirm no-dirty hooks
.PHONY: css css/watch css/install
.PHONY: build dev run
.PHONY: test test/cover test/integration lint fmt tidy audit vulncheck
.PHONY: docker docker/up docker/down
.PHONY: deploy deploy/backup deploy/rollback
.PHONY: changelog release release/dry
.PHONY: clean

BINARY  := yaitracker
PKG     := ./cmd/yaitracker
DB      := yaitracker.db
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# IDE/agent tasks often inherit a minimal PATH. If `go` is not already visible,
# prepend the directory of a Go binary from a standard install location, or /usr/bin:/bin.
ifeq ($(shell command -v go >/dev/null 2>&1 && echo ok),ok)
else
  _GO_BIN := $(firstword $(wildcard $(HOME)/go/bin/go /usr/local/go/bin/go /usr/lib/go/bin/go /usr/bin/go))
  ifneq ($(_GO_BIN),)
    export PATH := $(dir $(_GO_BIN)):$(PATH)
  else
    export PATH := /usr/bin:/bin:$(PATH)
  endif
endif

# ──────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────

## help: show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

no-dirty:
	@test -z "$$(git status --porcelain)" || (echo "error: working tree is dirty" && exit 1)

## hooks: install repo git hooks (conventional commits + pre-push vet/test)
hooks:
	@git config core.hooksPath .githooks
	@chmod +x .githooks/commit-msg .githooks/pre-push
	@echo "git hooks installed (core.hooksPath=.githooks). bypass: --no-verify or GITHOOKS_SKIP=1 on push"

# ──────────────────────────────────────────────────────
# CSS
# ──────────────────────────────────────────────────────

TAILWIND := ./tools/tailwindcss
CSS_IN   := static/css/input.css
CSS_OUT  := static/css/app.css

## css: build tailwind CSS (production, minified)
css:
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --minify

## css/watch: watch and rebuild CSS on changes
css/watch:
	$(TAILWIND) -i $(CSS_IN) -o $(CSS_OUT) --watch

## css/install: download the tailwind standalone CLI
css/install:
	@mkdir -p tools
	curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 -o $(TAILWIND)
	chmod +x $(TAILWIND)

# ──────────────────────────────────────────────────────
# Development
# ──────────────────────────────────────────────────────

## build: build CSS then compile the binary
build: css
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(PKG)

## dev: build and run with dev settings
dev: css build
	YAITRACKER_SECRET=dev-secret-must-be-at-least-32-chars-long \
	./$(BINARY) serve --db $(DB)

## run: build and run
run: build
	./$(BINARY) serve

# ──────────────────────────────────────────────────────
# Quality Control
# ──────────────────────────────────────────────────────

# Pinned with .github/workflows/ci.yml lint job; bump both together.
GOLANGCI_LINT_VER := v2.11.4
GOLANGCI_LINT     := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VER)

# Pinned with .github/workflows/ci.yml vulncheck job; bump both together.
# GOTOOLCHAIN must match go.mod when loading packages (see golang.org/x/vuln docs).
GOVULNCHECK_VER := v1.1.4

## test: run all unit tests with race detection
test:
	go test -v -race -count=1 -timeout=60s ./...

## test/cover: run tests with coverage report
test/cover:
	go test -v -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test/integration: run integration tests
test/integration:
	go test -v -race -count=1 -tags=integration ./...

## lint: run golangci-lint (same binary as CI; vs origin/master when present)
lint:
	@if git rev-parse --verify origin/master >/dev/null 2>&1; then \
		$(GOLANGCI_LINT) run --new-from-merge-base=origin/master; \
	else \
		$(GOLANGCI_LINT) run; \
	fi

## fmt: format code
fmt:
	gofmt -w .

## tidy: tidy modules
tidy:
	go mod tidy -v

## audit: run the full quality suite
audit: test lint
	go mod tidy -diff
	go mod verify
	go vet ./...
	$(MAKE) vulncheck

## vulncheck: check for known vulnerabilities
vulncheck:
	env GOTOOLCHAIN=$$(go env GOVERSION) go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VER) ./...

# ──────────────────────────────────────────────────────
# Docker
# ──────────────────────────────────────────────────────

## docker: build docker image
docker:
	docker build -t yaitracker:latest .

## docker/up: start services
docker/up:
	docker compose up -d

## docker/down: stop services
docker/down:
	docker compose down

# ──────────────────────────────────────────────────────
# Deploy (production)
# ──────────────────────────────────────────────────────

DEPLOY_ENV := deploy/.env
DEPLOY_SH  := deploy/deploy.sh

## deploy: build, transfer, and deploy to production (requires VERSION=vX.Y.Z)
deploy:
	@test -n "$(VERSION)" || (echo "error: VERSION is required, e.g. make deploy VERSION=v0.6.0" && exit 1)
	@test -f $(DEPLOY_ENV) || (echo "error: $(DEPLOY_ENV) not found -- copy deploy/.env.example and fill in values" && exit 1)
	$(DEPLOY_SH) $(VERSION)

## deploy/backup: back up the production database without deploying
deploy/backup:
	@test -f $(DEPLOY_ENV) || (echo "error: $(DEPLOY_ENV) not found" && exit 1)
	@set -a && . $(DEPLOY_ENV) && set +a && \
		TIMESTAMP=$$(date -u +%Y%m%d-%H%M%S) && \
		ssh "$$DEPLOY_HOST" "mkdir -p '$$DEPLOY_DATA_DIR/backups' && \
			cp '$$DEPLOY_DATA_DIR/yaitracker.db' '$$DEPLOY_DATA_DIR/backups/yaitracker-$$TIMESTAMP.db' && \
			[ -f '$$DEPLOY_DATA_DIR/yaitracker.db-wal' ] && cp '$$DEPLOY_DATA_DIR/yaitracker.db-wal' '$$DEPLOY_DATA_DIR/backups/yaitracker-$$TIMESTAMP.db-wal' || true" && \
		echo "Backup saved as yaitracker-$$TIMESTAMP.db"

## deploy/rollback: restore the most recent database backup
deploy/rollback:
	@test -f $(DEPLOY_ENV) || (echo "error: $(DEPLOY_ENV) not found" && exit 1)
	@set -a && . $(DEPLOY_ENV) && set +a && \
		LATEST=$$(ssh "$$DEPLOY_HOST" "ls -t '$$DEPLOY_DATA_DIR/backups'/yaitracker-*.db 2>/dev/null | head -1") && \
		[ -n "$$LATEST" ] || (echo "error: no backups found" && exit 1) && \
		echo "Restoring $$LATEST ..." && \
		ssh "$$DEPLOY_HOST" "docker stop yaitracker 2>/dev/null || true" && \
		ssh "$$DEPLOY_HOST" "cp '$$LATEST' '$$DEPLOY_DATA_DIR/yaitracker.db'" && \
		ssh "$$DEPLOY_HOST" "[ -f '$${LATEST}-wal' ] && cp '$${LATEST}-wal' '$$DEPLOY_DATA_DIR/yaitracker.db-wal' || true" && \
		ssh "$$DEPLOY_HOST" "docker start yaitracker" && \
		echo "Rollback complete."

# ──────────────────────────────────────────────────────
# Release
# ──────────────────────────────────────────────────────

## changelog: regenerate CHANGELOG.md from git history
changelog:
	git-cliff -o CHANGELOG.md

## release/dry: test goreleaser locally
release/dry:
	goreleaser release --snapshot --clean

## release: tag and push (triggers CI release)
release: confirm audit no-dirty
	@echo "Current version: $$(git describe --tags --abbrev=0 2>/dev/null || echo 'none')"
	@read -p "New version (e.g. v0.2.0): " ver && \
		git-cliff -o CHANGELOG.md && \
		git add CHANGELOG.md && \
		git commit -m "chore(release): prepare $$ver" && \
		git tag -a $$ver -m "release: $$ver" && \
		git push origin $$ver

# ──────────────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────────────

## clean: remove build artifacts
clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -f $(DB) $(DB)-wal $(DB)-shm
