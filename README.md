# RIP — IPv6 Proxy Pool Manager

A high-performance CLI tool for generating and managing IPv6 proxy pools with automatic health checking and 3proxy integration.

## Features

- **IPv6 Address Generation** — Generate unlimited IPv6 addresses from configurable prefixes
- **Proxy Pool Management** — Manage multiple proxy pools with different protocols (SOCKS5, HTTP)
- **Automatic Health Checking** — Monitor proxy health and automatically mark unhealthy proxies
- **3proxy Integration** — Generate 3proxy configurations for production deployment
- **Port Range Management** — Flexible port allocation with configurable ranges
- **Multiple Output Formats** — Support for JSON, YAML, and text output formats
- **Rate Limiting** — Built-in rate limiting to prevent API throttling

## Requirements

- Go 1.23.0 or later
- 3proxy (for production proxy servers)
- Linux/Unix/macOS/Windows

## Installation

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/nxhung1610/proxy-ipv6-generator-cli/main/install.sh | bash
```

This downloads the latest release binary for your platform (Linux/macOS/Windows, amd64/arm64) and installs it to `~/.local/bin/rip`.

> To also install 3proxy: `curl -fsSL ... | INSTALL_3PROXY=true bash`

### From Source

```bash
# Clone the repository
git clone https://github.com/nxhung/proxy-ipv6-generator-cli.git
cd proxy-ipv6-generator-cli

# Build the binary
make build

# Or install globally
sudo make install
```

### From Release

Download the latest binary for your platform from the releases page.

## Supported Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| Linux amd64 | Tested | Primary platform |
| Linux arm64 | Tested | Raspberry Pi, ARM servers |
| macOS amd64 | Tested | Intel Macs |
| macOS arm64 (Apple Silicon) | Tested | M1/M2/M3/M4 Macs |
| Windows | Partial | Basic functionality works; process management limited |

## Quick Start

### 1. Initialize Configuration

```bash
rip init
```

### 2. Configure Your IPv6 Prefix

Edit `~/.config/proxy-ipv6-cli/config.toml`:

```yaml
ipv6:
  prefix: "2001:db8::/32"
  interface: "eth0"

pools:
  default:
    portRange: "10000-10100"
    protocol: "socks5"
    healthCheck:
      enabled: true
      interval: "30s"
      timeout: "5s"
```

### 3. Generate IPv6 Addresses

```bash
# Generate 100 IPv6 addresses
rip generate --count 100

# Generate with custom prefix
rip generate --prefix 2001:db8:1::/48 --count 50
```

### 4. Start Proxy Pool

```bash
# Start the default pool
rip pool start

# Check status
rip status

# List all pools
rip pool list
```

### 5. Generate 3proxy Configuration

```bash
# Generate config for default pool
rip config generate

# Save to file
rip config generate --output /etc/3proxy/proxy.cfg

# Reload 3proxy
sudo systemctl reload 3proxy
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `rip init` | Initialize configuration |
| `rip generate` | Generate IPv6 addresses |
| `rip pool start` | Start proxy pool |
| `rip pool stop` | Stop proxy pool |
| `rip pool status` | Show pool status |
| `rip config generate` | Generate 3proxy config |
| `rip health check` | Run health check |
| `rip export` | Export proxies to file |
| `rip version` | Show version info |

For detailed command documentation, see [docs/cli-commands.md](docs/cli-commands.md).

## Configuration

Configuration is stored in `~/.config/proxy-ipv6-cli/config.toml`. See [docs/configuration.md](docs/configuration.md) for full configuration options.

## Architecture

For a detailed architecture overview, see [docs/architecture.md](docs/architecture.md).

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run tests with coverage
make test-coverage

# Lint code
make lint

# Build for all platforms
make build-all
```

## License

MIT License
