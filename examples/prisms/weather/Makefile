# Makefile for Weather Prism Example

BINARY_NAME=weather
INSTALL_DIR=$(HOME)/.local/bin

.PHONY: all build install clean run help

all: build

build:
	@echo "Building Weather prism..."
	@go mod tidy
	go build -o $(BINARY_NAME) .
	@echo "✓ Built: $(BINARY_NAME)"

install: build
	@echo "Installing to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY_NAME) $(INSTALL_DIR)/
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ Installed: $(INSTALL_DIR)/$(BINARY_NAME)"

run: build
	./$(BINARY_NAME)

clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "✓ Cleaned"

help:
	@echo "Weather Prism Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  make build   - Build the prism binary"
	@echo "  make install - Build and install to ~/.local/bin"
	@echo "  make run     - Build and run for testing"
	@echo "  make clean   - Remove build artifacts"
