# SPEC.md — RIP

## 1. Project Overview

**Name:** RIP — IPv6 Proxy Pool Manager
**Type:** Cross-platform CLI tool
**Binary name:** `rip`
**Core Function:** Generate, manage, and rotate IPv6 proxy servers backed by 3proxy  
**License:** MIT  
**Go Version:** 1.23+

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
| Language | Go 1.23+ | Single binary, cross-platform, fast |
| CLI Framework | Kong | DI via kong.Bind, subcommand-per-file pattern |
| Config | Viper + TOML | Env vars, flags, config file precedence |
| Logging | `log/slog` | Standard library, structured |
| Proxy Backend | 3proxy 0.9.x | De facto standard, 5K stars, native IPv6 |
| Build | `go build` + Makefile | Custom cross-platform release workflow |
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

Pool definitions are stored inline in `state.json` (see section 3.3). There is no separate `pools.json`.

---

## 4. CLI Command Tree

```
rip

  init                        # Create config file
  install                     # Install 3proxy (bundled/system/source)

  generate [flags]           # Generate IPv6 addresses
    --count N                # Number of addresses (default: from config or 100)
    --prefix N               # IPv6 prefix override (default: from config)
    --output FILE             # Output file (default: stdout, JSON)

  pool [subcommand]          # Pool management
    pool create <name> <prefix> <ports>  # Create a named pool
      --protocol N           # socks5 or http (default: socks5)
    pool list                # List all pools (JSON)
    pool remove <name>       # Remove a pool
      --yes                  # Skip confirmation
    pool start [name]        # Start a pool (default: "default")
    pool stop [name]         # Stop a pool (default: "default")
    pool status [name]       # Show pool status (default: first pool, JSON)

  config generate [flags]    # Generate 3proxy config for a pool
    --pool NAME              # Pool name (default: "default")
    --output FILE            # Output file (default: stdout)

  health check [flags]       # Run one-time health check
    --pool NAME              # Pool name (default: all proxies)

  export [flags]            # Export proxy list
    --format json|txt|csv   # Output format (default: json)
    --output FILE            # Output file (default: stdout)

  version                    # Print version info
```

**NOT YET IMPLEMENTED** (see §7 "Should Ship" / "Nice to Have"):
- `rip start`, `rip stop`, `rip status` (top-level)
- `rip rotate`, `rip logs`, `rip dashboard`, `rip completions`
- `rip pool scale`
- `rip config set/get/list`

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

TOML via [Viper](https://github.com/spf13/viper) (also handles env var binding and config precedence).

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

### Implemented
- [x] `rip init` creates a valid config file
- [x] `rip pool create` creates a named pool
- [x] `rip pool start` starts 3proxy with generated config for all proxies in the pool
- [x] `rip pool stop` cleanly terminates 3proxy
- [x] `rip pool status` shows pool status (JSON)
- [x] `rip pool list` lists all pools (JSON)
- [x] `rip pool remove` removes a pool
- [x] `rip generate` generates IPv6 addresses (JSON)
- [x] `rip config generate` produces a valid 3proxy.cfg for a pool
- [x] `rip health check` runs a one-time health check on proxies
- [x] `rip export --format json|txt|csv` produces correct output formats
- [x] `rip install` installs 3proxy (bundled → system → source)
- [x] `rip version` prints version info
- [x] Cross-platform binary builds for macOS, Linux, Windows (6 targets)
- [x] `--help` on every command
- [x] Kong CLI framework with DI via `kong.Bind`
- [x] Viper-based config with env var overrides
- [x] Health checking with blacklist
- [x] Multi-pool management (create, start, stop, remove)
- [x] SIGTERM/SIGKILL graceful shutdown

### Should Ship (not yet implemented)
- [ ] `rip start`, `rip stop`, `rip status` as top-level shortcuts
- [ ] `rip pool scale` to dynamically resize pools
- [ ] `rip config set/get/list` for runtime config editing
- [ ] IP rotation strategies (least-connections, time-based)
- [ ] `--version` with git info (ldflags injection)

### Nice to Have (not yet implemented)
- [ ] Web dashboard
- [ ] Bandwidth/connection limits
- [ ] Homebrew tap
- [ ] goreleaser release automation (current: custom shell workflow)

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
