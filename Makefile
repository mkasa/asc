.PHONY: build install uninstall clean

# Build variables
BINARY_NAME=asc
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Installation paths
PREFIX?=/usr/local
BINDIR=$(PREFIX)/bin

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/asc.go

# Install the application
install: build
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(BINDIR)
	@cp $(BINARY_NAME) $(BINDIR)/
	@chmod 755 $(BINDIR)/$(BINARY_NAME)
	@echo "Installed in $(BINDIR)/$(BINARY_NAME)"

# Uninstall the application
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(BINDIR)/$(BINARY_NAME)
	@echo "Uninstalled from $(BINDIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean

# Show help
help:
	@echo "Available targets:"
	@echo "  build     - Build the application"
	@echo "  install   - Install the application"
	@echo "  uninstall - Uninstall the application"
	@echo "  clean     - Clean build artifacts"
	@echo "  help      - Show this help message" 