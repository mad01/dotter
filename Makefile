BINARY_NAME=dotter
CMD_PATH=./cmd/dotter

.PHONY: all build test lint format clean

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "$(BINARY_NAME) built successfully."

test:
	@echo "Running tests..."
	@go test ./... -v

lint:
	@echo "Running linter (golangci-lint)..."
	@golangci-lint run ./...

format:
	@echo "Formatting code (goimports and gofmt)..."
	@goimports -w .
	@gofmt -w .

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@go clean

install_deps:
	@echo "Installing linter (golangci-lint) and formatter (goimports)..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

help:
	@echo "Available targets:"
	@echo "  all         - Build the binary (default)"
	@echo "  build       - Build the binary"
	@echo "  test        - Run unit tests"
	@echo "  lint        - Run golangci-lint (requires it to be installed)"
	@echo "  format      - Format code using goimports and gofmt"
	@echo "  clean       - Remove built binary and clean Go cache"
	@echo "  install_deps- Install development dependencies (golangci-lint, goimports)"
	@echo "  help        - Show this help message" 