# SPEC.md — RIP

## 1. Project Overview

**Name:** RIP — IPv6 Proxy Pool Manager
**Type:** Cross-platform CLI tool
**Binary name:** `rip`
**Core Function:** Generate, manage, and rotate IPv6 proxy servers backed by 3proxy  
**License:** MIT  
**Go Version:** 1.21+

### Motivation

IPv6 proxy tooling today is fragmented: shell scripts for Linux, Windows GUI tools, Docker configs. No modern, cross-platform CLI exists. Commercial proxies cost $0.25+/IP/month — self-hosting is cost-competitive for high volume.

### Scope

**In scope:**
- IPv6 address generation from /64 prefixes
- 3proxy process lifecycle (start/stop/restart/reload)
- Proxy pool management (create, scale, remove)
- IP rotation strategies (random, round-robin, sticky, least-connections, time-based)
- Health checking with automatic blacklist
- Credential management with external secret support
- Export in JSON, TXT, CSV formats
- Cross-platform: macOS, Linux, Windows

**Out of scope (YAGNI):**
- Built-in IPv6 subnet provisioning (Tunnelbroker, VPS APIs)
- Web scraping or proxy usage
- Commercial proxy API integration
- Cluster/multi-node deployments

---

## 2. Technology Stack

| Component | Choice | Reason |
|-----------|--------|--------|
| Language | Go 1.21+ | Single binary, cross-platform, fast |
| CLI Framework | Cobra | Standard, battle-tested |
| Config | Viper + TOML | Env vars, flags, config file precedence |
| Logging | `log/slog` | Standard library, structured |
| Proxy Backend | 3proxy 0.9.x | De facto standard, 5K stars, native IPv6 |
| Build | goreleaser | Cross-platform releases |
| CI | GitHub Actions | Free for public repos |

---

## 3. Data Model

### 3.1 Core Types (`pkg/types/types.go`)

```go
type Protocol string

const (
    ProtocolHTTP  Protocol = "http"
    ProtocolSOCKS5 Protocol = "socks5"
)

type RotationStrategy string

const (
    StrategyRandom      RotationStrategy = "random"
    StrategyRoundRobin  RotationStrategy = "round-robin"
    StrategySticky     RotationStrategy = "sticky"
    // Phase 9: StrategyLeastConn RotationStrategy = "least-connections"
    // Phase 9: StrategyTimeBased  RotationStrategy = "time-based"
)

type Proxy struct {
    ID       string   `json:"id"`
    Address  string   `json:"address"`  // "2001:db8::1"
    Port     int      `json:"port"`     // 10000
    Protocol Protocol `json:"protocol"` // http or socks5
    Username string   `json:"username"`
    Password string   `json:"password"` // never logged
    Status   string   `json:"status"`   // healthy, unhealthy, unknown
}

type Pool struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Prefix      string   `json:"prefix"`    // "2001:db8::/64"
    BasePort    int      `json:"basePort"`
    Count       int      `json:"count"`      // number of proxies
    Protocol    Protocol `json:"protocol"`
    Status      string   `json:"status"`    // running, stopped, error
    CreatedAt   string   `json:"createdAt"` // ISO 8601
    CredentialRef string  `json:"credentialRef"` // "env:MY_USER"
}
```

### 3.2 Configuration (`internal/config/config.go`)

Config file: TOML, auto-discovery order:
1. `./rip.toml` (project-local)
2. `~/.config/proxy-ipv6-cli/config.toml` (XDG macOS/Linux)
3. `%APPDATA%\proxy-ipv6-cli\config.toml` (Windows)

**Environment overrides:** All config keys overridable via `PROXY_IPV6_<KEY>` (uppercase, `_` separator).

```toml
[server]
prefix = "2001:db8::/64"       # Required: IPv6 prefix
basePort = 10000                # Default proxy start port
protocol = "socks5"             # "http" or "socks5"
count = 100                     # Default proxy count

[auth]
credentialRef = "env:MY_PROXY_USER"  # "env:USER:PASS" or "file:/path" or "inline:user:pass"

[proxy]
binaryPath = ""                 # Empty = auto-detect via PATH
maxConn = 500                  # Per-service connection limit
bandwidthLimitKBps = 0          # 0 = unlimited

[logging]
level = "info"                 # debug, info, warn, error
format = "text"                # text or json

[health]
enabled = true
intervalSeconds = 5
timeoutSeconds = 3
maxFailures = 3               # Blacklist after N consecutive failures
```

### 3.3 State File (`internal/state/state.go`)

Location: `~/.local/share/proxy-ipv6-cli/state.json` (macOS/Linux), `%LOCALAPPDATA%\proxy-ipv6-cli\state.json` (Windows).

```json
{
  "version": "0.1.0",
  "pools": [...],
  "3proxy": {
    "pid": 12345,
    "configPath": "/path/to/3proxy.cfg",
    "startedAt": "2026-05-09T19:50:00Z"
  }
}
```

### 3.4 Pool Registry

Location: `~/.local/share/proxy-ipv6-cli/pools.json`

```json
{
  "pools": [
    {
      "id": "uuid-v4",
      "name": "default",
      "prefix": "2001:db8::/64",
      "basePort": 10000,
      "count": 100,
      "protocol": "socks5",
      "credentialRef": "env:MY_PROXY_USER",
      "status": "running",
      "createdAt": "2026-05-09T19:50:00Z"
    }
  ]
}
```

---

## 4. CLI Command Tree

```
rip

  init                        # Interactive setup wizard
  start [flags]              # Start proxy pool
    --count N                 # Number of proxies (default: from config)
    --port N                  # Base port (default: from config)
    --pool NAME               # Pool name (default: "default")
    --dry-run                 # Preview without starting

  stop [flags]               # Stop proxy pool
    --pool NAME               # Pool name (default: "default")
    --force                   # SIGKILL instead of SIGTERM

  status [flags]             # Show proxy pool status
    --pool NAME               # Pool name (default: "default")
    --json                    # JSON output

  generate [flags]            # Generate IPv6 addresses
    --count N                 # Number of addresses (default: 10)
    --strategy random|sequential|range  # Generation strategy
    --start HEX               # Start offset (for sequential/range)
    --end HEX                 # End offset (for range)
    --seed N                  # RNG seed for reproducibility
    --format text|json|csv    # Output format (default: text)

  pool [subcommand]           # Pool management
    pool create [flags]       # Create a named pool
    pool list                 # List all pools
    pool remove NAME          # Remove a pool
    pool scale NAME --count N # Scale pool size

  config [subcommand]         # Config management
    config generate [flags]     # Generate 3proxy config [x] implemented
    config set KEY VALUE        # [ ] TODO — not yet implemented
    config get KEY              # [ ] TODO — not yet implemented
    config list                 # [ ] TODO — not yet implemented

  export [flags]              # Export proxy list
    --pool NAME               # Pool name (default: "default")
    --format json|txt|csv     # Output format (default: txt)
    --output FILE             # File path (default: stdout)

  rotate [flags]              # Rotate IPs (manual trigger)
    --pool NAME               # Pool name (default: "default")
    --strategy NAME           # Override rotation strategy

  health [subcommand]         # Health check management
    health check              # Run health checks once
    health list               # List all proxy health statuses
    health clear IP:PORT      # Remove from blacklist

  logs [subcommand]           # Log management
    logs tail                 # Stream live logs
    logs stats [flags]        # Show aggregate statistics
      --since DURATION        # e.g. "1h", "24h"
    logs export [flags]       # Export logs
      --format json|txt
      --since DURATION

  dashboard [flags]           # Start web dashboard
    --port N                  # Dashboard port (default: 8080)
    --auth USER:PASS          # Basic auth (default: none)
    --detach                  # Run in background

  version                     # Print version info
  completions [shell]          # Generate shell completions
```

---

## 5. Error Handling

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | 3proxy binary not found |
| 4 | Port already in use |
| 5 | Pool not found |
| 6 | Permission denied |

### Structured Errors (`internal/errs/errors.go`)

All errors wrap one of:
```go
var (
    ErrNoPrefix       = errors.New("ipv6_prefix_required")
    ErrInvalidPrefix  = errors.New("invalid_ipv6_prefix")
    Err3proxyNotFound = errors.New("3proxy_binary_not_found")
    ErrPortInUse      = errors.New("port_already_in_use")
    ErrPoolNotFound   = errors.New("pool_not_found")
    ErrPoolExists     = errors.New("pool_already_exists")
    ErrInvalidPort    = errors.New("invalid_port_range")
    ErrPermDenied     = errors.New("permission_denied")
    ErrConfigNotFound = errors.New("config_file_not_found")
)
```

---

## 6. Configuration File Format

TOML via [BurntSushi/toml](https://github.com/BurntSushi/toml).

All keys are optional except `server.prefix`. Missing keys use defaults.

**Example `rip.toml`:**
```toml
[server]
prefix = "2001:db8:1:4::/64"
basePort = 10000
protocol = "socks5"
count = 100

[auth]
credentialRef = "env:MY_PROXY_USER"

[proxy]
maxConn = 500
bandwidthLimitKBps = 0

[logging]
level = "info"
format = "text"

[health]
enabled = true
intervalSeconds = 5
timeoutSeconds = 3
maxFailures = 3
```

---

## 7. Acceptance Criteria

### Must Ship (MVP)
- [ ] `rip init` creates a valid config file
- [ ] `rip start --count 100` starts 100 proxies, each with a unique IPv6 address
- [ ] `rip status` shows accurate running/stopped state
- [ ] `rip generate --count 1000` outputs 1000 unique IPv6 addresses
- [ ] `rip stop` cleanly terminates 3proxy
- [ ] `rip export --format json` produces valid JSON
- [ ] Cross-platform binary builds for macOS, Linux, Windows
- [ ] `--help` on every command with examples

### Should Ship
- [ ] Health checking with blacklist
- [ ] Rotation strategies
- [ ] Multi-pool management
- [ ] Structured JSON logging
- [ ] `--version` with git info

### Nice to Have
- [ ] Web dashboard
- [ ] Bandwidth/connection limits
- [ ] Homebrew tap
- [ ] goreleaser release automation

---

## 8. Constraints

- **Binary size:** < 10MB
- **Startup time:** CLI commands respond in < 100ms (except start/stop)
- **No external runtime deps** — single static binary
- **IPv6 connectivity required on host** — tool does not provision subnets

---

## 9. 3proxy Bundling & Auto-Install

### Bundled Binaries

3proxy binaries (v0.9.6) are bundled in `3proxy-bin/` alongside the release binary:

```
rip/
├── rip              (or rip.exe on Windows)
└── 3proxy-bin/
    ├── 3proxy-linux-amd64      # Linux x86-64
    ├── 3proxy-linux-arm64       # Linux ARM64
    ├── 3proxy-windows-amd64.exe # Windows x86-64
    └── 3proxy-windows-arm64.exe # Windows ARM64
    # (macOS binaries not available from 3proxy project; falls back to PATH)
```

Binary discovery order on `rip pool start` / `rip install`:

1. **Bundled** — `$(rip_dir)/3proxy-bin/3proxy-<os>-<arch>` (Linux/Windows only)
2. **User-local** — `~/.local/share/proxy-ipv6-cli/bin/3proxy` (Linux/macOS), `%LOCALAPPDATA%\proxy-ipv6-cli\bin\3proxy.exe` (Windows)
3. **System** — `/usr/local/bin/3proxy`, `/usr/bin/3proxy`, `/opt/3proxy/3proxy`
4. **PATH** — `3proxy` via `$PATH` lookup

### `rip install`

Downloads/copies the bundled 3proxy binary to the user-local install path:

```bash
# On Linux/Windows: copies bundled binary
rip install

# On macOS: copies from system PATH (or instructs user to compile from source)
rip install
```

### Pool Lifecycle

`rip pool start` now fully manages the 3proxy process:

1. Detects 3proxy binary (bundled → local → system)
2. Generates `3proxy.cfg` for the pool
3. Writes config to `~/.local/share/proxy-ipv6-cli/pools/<pool-id>/3proxy.cfg`
4. Spawns `3proxy <pool-id>/3proxy.cfg`
5. Writes PID to `~/.local/share/proxy-ipv6-cli/pools/<pool-id>/3proxy.pid`

`rip pool stop` sends SIGTERM/SIGKILL, removes PID file, updates state.
