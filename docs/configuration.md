# Configuration Guide

## Config File Location

The default config file is located at:
- **Linux/macOS:** `~/.config/proxy-ipv6-cli/config.toml`
- **Windows:** `%APPDATA%\proxy-ipv6-cli\config.toml`

The tool searches for config files in this order:
1. `./rip.toml` (project-local)
2. `~/.config/proxy-ipv6-cli/config.toml` (XDG macOS/Linux)
3. `%APPDATA%\proxy-ipv6-cli\config.toml` (Windows)

You can also specify a custom config path using the `--config` or `-c` flag on any command.

---

## Installation

### Install Script

For quick installation, use the official install script:

```bash
curl -fsSL https://raw.githubusercontent.com/nxhung1610/proxy-ipv6-generator-cli/main/install.sh | bash
```

This downloads the latest release binary for your platform and installs it to `~/.local/bin/rip`.

**Install with 3proxy:**

```bash
curl -fsSL https://raw.githubusercontent.com/nxhung1610/proxy-ipv6-generator-cli/main/install.sh | INSTALL_3PROXY=true bash
```

### Supported Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| Linux amd64 | Tested | Primary platform |
| Linux arm64 | Tested | Raspberry Pi, ARM servers |
| macOS amd64 | Tested | Intel Macs |
| macOS arm64 (Apple Silicon) | Tested | M1/M2/M3/M4 Macs |
| WSL (Windows Subsystem for Linux) | Tested | Uses Linux binary natively |
| Windows | Partial | Basic functionality; process management limited |

---

## Config File Format

The config file uses TOML format. All keys are optional except `server.prefix` which is required.

```toml
[server]
prefix = "2001:db8::/64"       # Required: IPv6 prefix in CIDR notation
basePort = 10000                # Default: 10000
protocol = "socks5"             # Options: "socks5" or "http" (default: socks5)
count = 100                     # Default proxy count (default: 100)

[auth]
credentialRef = ""               # Credential source: "env:USER:PASS", "file:/path", or inline:user:pass

[proxy]
binaryPath = ""                 # Empty = auto-detect via PATH
maxConn = 500                   # Per-service connection limit (default: 500)
bandwidthLimitKBps = 0          # Per-proxy bandwidth limit in KB/s (0 = unlimited)

[logging]
level = "info"                  # Options: debug, info, warn, error (default: info)
format = "text"                 # Options: "text" or "json" (default: text)

[health]
enabled = true                  # Enable health checking (default: true)
intervalSeconds = 5              # Check interval in seconds (default: 5)
timeoutSeconds = 3               # Check timeout in seconds (default: 3)
maxFailures = 3                 # Consecutive failures before blacklist (default: 3)
```

---

## Environment Variables

All config values can be overridden with environment variables using the `PROXY_IPV6_` prefix.

| Config Key | Environment Variable | Example |
|-----------|---------------------|---------|
| `server.prefix` | `PROXY_IPV6_SERVER_PREFIX` | `2001:db8::/64` |
| `server.basePort` | `PROXY_IPV6_SERVER_BASE_PORT` | `10000` |
| `server.protocol` | `PROXY_IPV6_SERVER_PROTOCOL` | `socks5` |
| `server.count` | `PROXY_IPV6_SERVER_COUNT` | `100` |
| `auth.credentialRef` | `PROXY_IPV6_AUTH_CREDENTIAL_REF` | `env:MY_CREDS` |
| `proxy.binaryPath` | `PROXY_IPV6_PROXY_BINARY_PATH` | `/usr/local/bin/3proxy` |
| `proxy.maxConn` | `PROXY_IPV6_PROXY_MAX_CONN` | `1000` |
| `proxy.bandwidthLimitKBps` | `PROXY_IPV6_PROXY_BANDWIDTH_LIMIT_KBPS` | `5120` |
| `logging.level` | `PROXY_IPV6_LOGGING_LEVEL` | `debug` |
| `logging.format` | `PROXY_IPV6_LOGGING_FORMAT` | `json` |
| `health.enabled` | `PROXY_IPV6_HEALTH_ENABLED` | `true` |
| `health.intervalSeconds` | `PROXY_IPV6_HEALTH_INTERVAL_SECONDS` | `10` |
| `health.timeoutSeconds` | `PROXY_IPV6_HEALTH_TIMEOUT_SECONDS` | `5` |
| `health.maxFailures` | `PROXY_IPV6_HEALTH_MAX_FAILURES` | `5` |

---

## Example Configurations

### Minimal Setup

```toml
[server]
prefix = "2001:db8::/64"
```

### Production Setup

```toml
[server]
prefix = "2001:db8:1::/64"
basePort = 10000
protocol = "socks5"
count = 500

[auth]
credentialRef = "env:MY_PROXY_CREDS"

[proxy]
maxConn = 1000
bandwidthLimitKBps = 0

[logging]
level = "info"
format = "text"

[health]
enabled = true
intervalSeconds = 10
timeoutSeconds = 5
maxFailures = 3
```

---

## Data Directory

The tool stores runtime data in:
- **Linux/macOS:** `~/.local/share/proxy-ipv6-cli/`
- **Windows:** `%LOCALAPPDATA%\proxy-ipv6-cli\`

This directory contains:
- `state.json` — Process state and PID
- `pools/` — Per-pool config and PID files

---

## See Also

- [CLI Commands](./cli-commands.md)
- [Architecture](./architecture.md)
