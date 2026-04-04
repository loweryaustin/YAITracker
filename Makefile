.PHONY: help confirm no-dirty
.PHONY: build dev run
.PHONY: test test/cover test/integration lint fmt tidy audit vulncheck
.PHONY: docker docker/up docker/down
.PHONY: changelog release release/dry
.PHONY: clean

BINARY  := yaitracker
PKG     := ./cmd/yaitracker
DB      := yaitracker.db
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

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

# ──────────────────────────────────────────────────────
# Development
# ──────────────────────────────────────────────────────

## build: compile the binary
build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(PKG)

## dev: build and run with dev settings
dev: build
	YAITRACKER_SECRET=dev-secret-must-be-at-least-32-chars-long \
	./$(BINARY) serve --db $(DB)

## run: build and run
run: build
	./$(BINARY) serve

# ──────────────────────────────────────────────────────
# Quality Control
# ──────────────────────────────────────────────────────

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

## lint: run golangci-lint
lint:
	golangci-lint run

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
	govulncheck ./...

## vulncheck: check for known vulnerabilities
vulncheck:
	govulncheck ./...

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
