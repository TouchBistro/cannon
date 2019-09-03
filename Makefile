.DEFAULT_GOAL = build

# Get all dependencies
setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	go mod download
.PHONY: setup

# Build tb
build:
	go build
.PHONY: build

# Clean all build artifacts
clean:
	rm cannon
.PHONY: clean

# Run the linter
lint:
	./bin/golangci-lint run ./...
.PHONY: lint

# Remove version of cannon installed with go install
go-uninstall:
	rm $(shell go env GOPATH)/bin/cannon
.PHONY: go-uninstall
