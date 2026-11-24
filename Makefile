.PHONY: run demo reset clean build test check lint lint-code vulncheck help web-build web-check web-types

CONFIG ?= config.yaml
# The version built into the binary: the latest tag (v0.3.0), or the tag plus
# the commits since it and the commit hash (v0.3.0-4-g1a2b3c4), with -dirty
# when the tree has uncommitted changes. Without tags it is just the commit.
# Override with make build VERSION=v0.3.0.
VERSION ?= $(shell git describe --tags --match "v*" --always --dirty 2>/dev/null || echo dev)
BIN := household-app$(shell go env GOEXE)

help: ## Show available commands
	@echo "Available commands:"
	@echo "  make run   - Build the frontend and the server, then start it (one port, API and UI)"
	@echo "  make demo  - Build, then run on an invented demo household (port 8081, user alex or sam, password demo)"
	@echo "  make reset - Delete the database, then run"
	@echo "  make clean - Delete the database"
	@echo "  make build - Build only the server binary (household-app) with the version set"
	@echo "  make test  - Run the tests, shuffled"
	@echo "  make check - Run the local format, build, vet and test checks"
	@echo "  make lint  - Run golangci-lint and govulncheck locally (needs network)"
	@echo "  make lint-code - Run golangci-lint only (needs network)"
	@echo "  make vulncheck - Run govulncheck only (needs network)"
	@echo "  make web-build - Build the frontend into web/dist"
	@echo "  make web-check - Frontend format check, types, tests and build, as CI runs them"
	@echo "  make web-types - Regenerate the frontend API types from docs/architecture/openapi.yaml"

run: web-build build ## Build the frontend and the server, then start it (it prepares the data itself)
	./$(BIN) -config $(CONFIG)

demo: web-build build ## Run on a copy of the invented demo household in demo/ (port 8081)
	./scripts/demo.sh ./$(BIN)

reset: clean run ## Delete database, then run

clean: ## Delete the database
	rm -f household.db household.db-wal household.db-shm

build: ## Build only the server binary with the version set (embeds the last frontend build)
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN) ./cmd/server/

test: ## Run the tests in random order
	go test -shuffle=on ./...

check: ## Format check, build, vet and tests
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	go build ./...
	go vet ./...
	go test -shuffle=on ./...

GOLANGCI_LINT = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0

lint: lint-code vulncheck ## golangci-lint and govulncheck (needs network)

lint-code:
	$(GOLANGCI_LINT) run ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# The frontend dependencies are installed when missing or when the lock file
# changes (needs Node, see web/.nvmrc).
web/node_modules: web/package-lock.json
	npm --prefix web ci
	@touch web/node_modules

web-build: web/node_modules ## Build the frontend into web/dist (the server embeds it)
	npm --prefix web run build

web-check: web/node_modules ## Frontend format check, types, tests and build, as CI runs them
	npm --prefix web run format:check
	npm --prefix web run check
	npm --prefix web test
	npm --prefix web run build

web-types: web/node_modules ## Regenerate the frontend API types from docs/architecture/openapi.yaml
	npm --prefix web run api:types
