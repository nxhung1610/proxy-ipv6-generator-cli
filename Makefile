.PHONY: build test lint clean install run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOLINT=golangci-lint

# Binary name
BINARY_NAME=rip
BINARY_PATH=bin/$(BINARY_NAME)

# Go files
GOFILES=$(shell find . -name '*.go' -not -path './.venv/*' -not -path './.git/*')

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/cli

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Lint the code
lint:
	@echo "Linting code..."
	$(GOVET) ./...
	@which $(GOLINT) > /dev/null && $(GOLINT) run || echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/ 3proxy-bin/
	@rm -f coverage.out coverage.html

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Verify dependencies
verify:
	$(GOMOD) verify

# Install the binary
install:
	$(GOBUILD) -o /usr/local/bin/$(BINARY_NAME) ./cmd/cli

# Run the CLI (development)
run:
	@echo "Running CLI..."
	$(GOBUILD) -o /tmp/$(BINARY_NAME) ./cmd/cli && /tmp/$(BINARY_NAME)

# Download 3proxy binaries (all platforms) into 3proxy-bin/
vendor-3proxy:
	@echo "Downloading 3proxy binaries..."
	@mkdir -p 3proxy-bin
# Linux amd64 (extract from .deb via bsdtar or zstd)
	@if command -v dpkg &>/dev/null; then \
		curl -sL "https://github.com/3proxy/3proxy/releases/download/0.9.6/3proxy-0.9.6.x86_64.deb" -o /tmp/3proxy-amd64.deb && \
		dpkg -x /tmp/3proxy-amd64.deb /tmp/3proxy-amd64 && \
		mv /tmp/3proxy-amd64/usr/bin/3proxy 3proxy-bin/3proxy-linux-amd64 && \
		chmod +x 3proxy-bin/3proxy-linux-amd64 && \
		rm -rf /tmp/3proxy-amd64 /tmp/3proxy-amd64.deb; \
	else \
		curl -sL "https://github.com/3proxy/3proxy/releases/download/0.9.6/3proxy-0.9.6.x86_64.deb" -o /tmp/3proxy-amd64.deb && \
		bsdtar -xf /tmp/3proxy-amd64.deb data.tar.zst 2>/dev/null || zstd -d /tmp/3proxy-amd64.deb 2>/dev/null || true && \
		bsdtar -xf /tmp/3proxy-amd64.deb data.tar.zst && \
		bsdtar -xf data.tar.zst && \
		mv usr/bin/3proxy 3proxy-bin/3proxy-linux-amd64 && chmod +x 3proxy-bin/3proxy-linux-amd64 && \
		rm -rf usr etc debian-binary control.tar.zst data.tar.zst /tmp/3proxy-amd64.deb; \
	fi
# Linux arm64
	@curl -sL "https://github.com/3proxy/3proxy/releases/download/0.9.6/3proxy-0.9.6.arm64.deb" -o /tmp/3proxy-arm64.deb && \
		bsdtar -xf /tmp/3proxy-arm64.deb data.tar.zst && \
		bsdtar -xf data.tar.zst && \
		mv usr/bin/3proxy 3proxy-bin/3proxy-linux-arm64 && chmod +x 3proxy-bin/3proxy-linux-arm64 && \
		rm -rf usr etc debian-binary control.tar.zst data.tar.zst /tmp/3proxy-arm64.deb
# Windows amd64
	@curl -sL "https://github.com/3proxy/3proxy/releases/download/0.9.6/3proxy-0.9.6.1-x64.zip" -o /tmp/3proxy-win.zip && \
		unzip -o /tmp/3proxy-win.zip "3proxy/bin64/3proxy.exe" && \
		mv 3proxy/bin64/3proxy.exe 3proxy-bin/3proxy-windows-amd64.exe && \
		rm -rf 3proxy /tmp/3proxy-win.zip
# Windows arm64
	@curl -sL "https://github.com/3proxy/3proxy/releases/download/0.9.6/3proxy-0.9.6.1-arm64.zip" -o /tmp/3proxy-win-arm64.zip && \
		unzip -o /tmp/3proxy-win-arm64.zip "3proxy/bin64/3proxy.exe" && \
		mv 3proxy/bin64/3proxy.exe 3proxy-bin/3proxy-windows-arm64.exe && \
		rm -rf 3proxy /tmp/3proxy-win-arm64.zip
	@echo "3proxy binaries downloaded to 3proxy-bin/"

# Build for all platforms
build-all: vendor-3proxy
	@mkdir -p bin
# Linux amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="-s -w" -o bin/rip-linux-amd64 ./cmd/cli
	@mkdir -p bin/rip-linux-amd64
	cp bin/rip-linux-amd64 bin/rip-linux-amd64/rip
	cp 3proxy-bin/3proxy-linux-amd64 bin/rip-linux-amd64/3proxy
# Linux arm64
	GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags="-s -w" -o bin/rip-linux-arm64 ./cmd/cli
	@mkdir -p bin/rip-linux-arm64
	cp bin/rip-linux-arm64 bin/rip-linux-arm64/rip
	cp 3proxy-bin/3proxy-linux-arm64 bin/rip-linux-arm64/3proxy
# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags="-s -w" -o bin/rip-darwin-amd64 ./cmd/cli
	@mkdir -p bin/rip-darwin-amd64
	cp bin/rip-darwin-amd64 bin/rip-darwin-amd64/rip
	@if [ -f 3proxy-bin/3proxy-darwin-amd64 ]; then cp 3proxy-bin/3proxy-darwin-amd64 bin/rip-darwin-amd64/3proxy; else echo "Note: no macOS 3proxy binary (build on Linux with vendor-3proxy)"; fi
# macOS arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags="-s -w" -o bin/rip-darwin-arm64 ./cmd/cli
	@mkdir -p bin/rip-darwin-arm64
	cp bin/rip-darwin-arm64 bin/rip-darwin-arm64/rip
	@if [ -f 3proxy-bin/3proxy-darwin-arm64 ]; then cp 3proxy-bin/3proxy-darwin-arm64 bin/rip-darwin-arm64/3proxy; else echo "Note: no macOS 3proxy binary (build on Linux with vendor-3proxy)"; fi
# Windows amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags="-s -w" -o bin/rip-windows-amd64.exe ./cmd/cli
	@mkdir -p bin/rip-windows-amd64
	cp bin/rip-windows-amd64.exe bin/rip-windows-amd64/rip.exe
	cp 3proxy-bin/3proxy-windows-amd64.exe bin/rip-windows-amd64/3proxy.exe
# Windows arm64
	GOOS=windows GOARCH=arm64 $(GOBUILD) -ldflags="-s -w" -o bin/rip-windows-arm64.exe ./cmd/cli
	@mkdir -p bin/rip-windows-arm64
	cp bin/rip-windows-arm64.exe bin/rip-windows-arm64/rip.exe
	cp 3proxy-bin/3proxy-windows-arm64.exe bin/rip-windows-arm64/3proxy.exe
	@echo "Build complete. Binaries in bin/"
	@echo "  bin/rip-linux-amd64/   - Linux x86-64"
	@echo "  bin/rip-linux-arm64/   - Linux ARM64"
	@echo "  bin/rip-darwin-amd64/  - macOS Intel"
	@echo "  bin/rip-darwin-arm64/  - macOS Apple Silicon"
	@echo "  bin/rip-windows-amd64/ - Windows x86-64"
	@echo "  bin/rip-windows-arm64/ - Windows ARM64"

# Generate 3proxy config example
config-gen:
	$(GOCMD) run ./cmd/cli config generate --pool default

# Start a proxy pool
pool-start:
	$(GOCMD) run ./cmd/cli pool start

# Stop the proxy pool
pool-stop:
	$(GOCMD) run ./cmd/cli pool stop

# Show pool status
pool-status:
	$(GOCMD) run ./cmd/cli pool status

# Health check all proxies
health-check:
	$(GOCMD) run ./cmd/cli health check

# Generate IPv6 addresses
generate:
	$(GOCMD) run ./cmd/cli generate

# Show version
version:
	$(GOCMD) run ./cmd/cli version
