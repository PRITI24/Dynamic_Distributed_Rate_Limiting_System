.PHONY: build run test clean

# Build the application
build:
	go build -o bin/ratelimiter cmd/ratelimiter/main.go

# Run the application
run:
	go run cmd/ratelimiter/main.go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Run linter
lint:
	golangci-lint run

# Generate mocks (if needed)
mocks:
	mockgen -source=internal/ratelimiter/ratelimiter.go -destination=internal/mocks/ratelimiter_mock.go

# Download dependencies
deps:
	go mod download

.DEFAULT_GOAL := build 