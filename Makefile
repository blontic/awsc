.PHONY: build clean test test-verbose test-coverage run deps install build-all

# Build the binary
build:
	go build -o swa main.go

# Build for multiple platforms
build-all:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/swa-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/swa-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -o bin/swa-linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -o bin/swa-windows-amd64.exe main.go

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

# Install the tool to GOPATH/bin
install:
	go install

# Development workflow: build and test
dev: deps test build
	@echo "Development build complete"