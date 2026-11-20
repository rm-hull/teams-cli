.PHONY: build test lint clean

# Default target
all: build test lint

# Build the application
build:
	go build -o teams-cli main.go

# Run tests
test:
	go test -v -coverprofile=profile.cov -coverpkg=./... ./...

# Run tests in CI
test-ci:
	mkdir -p ./test-reports/
	gotestsum --junitfile=./test-reports/junit.xml --format github-actions -- -v -coverprofile=profile.cov -coverpkg=./... ./...

# Run linting
lint:
	golangci-lint run ./...

# Clean up build artifacts
clean:
	rm -f teams-cli profile.cov
	rm -rf ./test-reports/