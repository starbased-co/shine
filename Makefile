.PHONY: all build test clean prismctl prisms test-prism help

# Default target
all: build test-prism

# Build everything
build: prismctl prisms

# Build prismctl supervisor
prismctl:
	@echo "Building prismctl..."
	@go build -o bin/prismctl ./cmd/prismctl

# Build all example prisms
prisms:
	@echo "Building example prisms..."
	@go build -o bin/shine ./cmd/shine
	@go build -o bin/shinectl ./cmd/shinectl
	@go build -o bin/shine-clock ./cmd/shine-clock
	@go build -o bin/shine-chat ./cmd/shine-chat
	@go build -o bin/shine-bar ./cmd/shine-bar
	@go build -o bin/shine-sysinfo ./cmd/shine-sysinfo

# Build test fixture prism
test-prism:
	@echo "Building test-prism fixture..."
	@cd test/fixtures && go build -o test-prism test_prism.go

# Run unit tests
test:
	@echo "Running unit tests..."
	@go test ./cmd/... ./pkg/...

# Run integration tests (requires PTY)
test-integration:
	@echo "Running integration tests..."
	@cd test/integration && go test -v

# Run all tests
test-all: test test-integration

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f bin/prismctl bin/shine bin/shinectl
	@rm -f bin/shine-*
	@rm -f test/fixtures/test-prism

# Quick test - builds binaries and shows how to run
quick-test: prismctl test-prism
	@echo ""
	@echo "=== Quick Test Ready ==="
	@echo "Binaries built successfully!"
	@echo ""
	@echo "To test, run this command in your terminal:"
	@echo "  ./bin/prismctl panel-test ./test/fixtures/test-prism"
	@echo ""
	@echo "Note: prismctl requires a real TTY and cannot run through Make."
	@echo ""

# Install to system (optional)
install: build
	@echo "Installing to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp bin/prismctl ~/.local/bin/
	@cp bin/shine* ~/.local/bin/
	@echo "Done! Make sure ~/.local/bin is in your PATH"

# Help
help:
	@echo "Shine Build Commands:"
	@echo ""
	@echo "  make              - Build prismctl + test-prism"
	@echo "  make build        - Build everything (prismctl + all prisms)"
	@echo "  make prismctl     - Build only prismctl"
	@echo "  make prisms       - Build only example prisms"
	@echo "  make test-prism   - Build test fixture"
	@echo ""
	@echo "  make test         - Run unit tests"
	@echo "  make test-all     - Run all tests (unit + integration)"
	@echo "  make quick-test   - Build and launch test-prism"
	@echo ""
	@echo "  make clean        - Remove build artifacts"
	@echo "  make install      - Install to ~/.local/bin"
	@echo "  make help         - Show this help"
