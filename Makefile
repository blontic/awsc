.PHONY: build clean test test-verbose test-coverage run deps install build-all fmt mocks

# Version variables
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -s -w -X github.com/blontic/swa/cmd.Version=$(VERSION) -X github.com/blontic/swa/cmd.Commit=$(COMMIT) -X github.com/blontic/swa/cmd.Date=$(DATE)

# Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o swa main.go

# Build for multiple platforms
build-all:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/swa-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/swa-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/swa-linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/swa-windows-amd64.exe main.go

# Clean build artifacts
clean:
	rm -rf bin/ swa

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run the tool
run:
	go run main.go

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format all Go code
fmt:
	go fmt ./...

# Install the tool to GOPATH/bin
install:
	go install

# Generate mocks for testing
mocks:
	rm -rf internal/aws/mocks
	mkdir -p internal/aws/mocks
	cd internal/aws && go run go.uber.org/mock/mockgen -destination=mocks/aws_mocks.go -package=mocks . RDSClient,EC2Client,SSMClient,SecretsManagerClient

# Development workflow: build and test
dev: mocks deps test build
	@echo "Development build complete"