.DEFAULT_GOAL = build

# Get all dependencies
setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	go mod download
.PHONY: setup

# Build cannon
build:
	go build
.PHONY: build

# Clean all build artifacts
clean:
	rm cannon
	rm -rf coverage
.PHONY: clean

# Run the linter
lint:
	./bin/golangci-lint run ./...
.PHONY: lint

# Remove version of cannon installed with go install
go-uninstall:
	rm $(shell go env GOPATH)/bin/cannon
.PHONY: go-uninstall

# Run tests and collect coverage data
test:
	mkdir -p coverage
	go test -coverprofile=coverage/coverage.txt ./...
	go tool cover -html=coverage/coverage.txt -o coverage/coverage.html
.PHONY: test
