.PHONY: setup build test vet lint fmt ci

setup: ## One-time dev environment setup: git hooks + deps
	git config core.hooksPath .githooks
	go mod download

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: ## Requires golangci-lint (https://golangci-lint.run/welcome/install)
	golangci-lint run

ci: ## What CI runs, for a quick local check before pushing
	gofmt -l . | (! grep .)
	go vet ./...
	go build ./...
	go test ./... -race -shuffle=on
