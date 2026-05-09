# RIP — Architecture

## System Overview

```
User CLI (Kong)
    │
    ▼
Config Loader (Viper + TOML)
    │  env vars, flags, config file
    ▼
Command Handlers (cmd/*)
    │
    ├──► IPv6 Generator (internal/generator)
    │         │
    │         ▼
    ├──► 3proxy Config Renderer (internal/proxy/config.go)
    │         │  text/template from configs/3proxy.cfg.tmpl
    │         ▼
    ├──► 3proxy Process Manager (internal/proxy/manager.go)
    │         │  PID file, start/stop/restart
    │         ▼
    │    3proxy Process (external binary)
    │
    ├──► Pool Manager (internal/pool)
    │         │  Multi-pool registry, state.json
    │         ▼
    ├──► Credential Provider (internal/credential)
    │         │  env: / file: / inline:
    │         ▼
    ├──► Health Checker (internal/health)
    │         │  Background goroutine, blacklist
    │         ▼
    ├──► Rotator (internal/rotator)
    │         │  Strategy selection
    │         ▼
    └──► Dashboard (internal/dashboard)
              │  HTTP server, embedded HTML
              ▼
           HTTP (optional)
```

## Components

### `cmd/` — CLI Commands

Cobra command tree. Each subcommand is a separate file.

```
cmd/
├── root.go              # Root command, config loading, logging init
├── version.go           # version subcommand
├── init.go              # Interactive init wizard
├── start.go             # Start proxy pool
├── stop.go              # Stop proxy pool
├── status.go            # Status report
├── generate.go          # IPv6 address generation
├── export.go            # Export proxy list
├── rotate.go            # Manual IP rotation
├── pool/
│   ├── pool.go          # Parent command
│   ├── create.go        # pool create
│   ├── list.go          # pool list
│   ├── remove.go        # pool remove
│   └── scale.go         # pool scale
├── config/
│   ├── config.go        # Parent command
│   ├── set.go           # config set
│   ├── get.go           # config get
│   └── list.go          # config list
├── health/
│   ├── health.go         # Parent command
│   ├── check.go         # health check
│   ├── list.go          # health list
│   └── clear.go         # health clear
├── logs/
│   ├── logs.go          # Parent command
│   ├── tail.go          # logs tail
│   ├── stats.go         # logs stats
│   └── export.go        # logs export
└── dashboard.go          # dashboard command
```

### `internal/generator/` — IPv6 Address Generation

Generates IPv6 addresses from a /64 prefix. Three strategies: random, sequential, range.

```
generator.go      # Interface + NewGenerator
random.go         # crypto/rand based IID generation
sequential.go     # Sequential from start offset
range.go          # Bounded range generation
options.go        # Generator options
```

**Key interface:**
```go
type Generator interface {
    Next() (net.IP, error)
    Batch(n int) []net.IP
    Count() int64
    Reset()
}
```

### `internal/proxy/` — 3proxy Integration

Manages 3proxy process and config generation.

```
manager.go    # Process lifecycle (start/stop/restart/reload)
config.go     # 3proxy.cfg generation via text/template
users.go      # users.conf management
ndppd.go      # NDP proxy config (optional, Linux)
```

**3proxy config flow:**
1. Generate N IPv6 addresses from pool prefix
2. Create N users (auto-generated credentials)
3. Render 3proxy.cfg template with N `proxy -6 -p{PORT} -e{IP}` entries
4. Write users.conf
5. Spawn 3proxy process
6. Write PID to state file

### `internal/pool/` — Pool Management

Multi-pool registry with persistent state.

```
manager.go    # Multi-pool orchestration
registry.go   # pools.json read/write
scale.go      # Atomic scale up/down
```

### `internal/health/` — Health Checking

Background health probe with automatic blacklist.

```
checker.go       # Health check engine (background goroutine)
http_probe.go    # HTTP CONNECT probe
socks5_probe.go  # SOCKS5 handshake probe
blacklist.go     # Unhealthy proxy tracking
```

### `internal/rotator/` — IP Rotation

Strategy-based proxy selection.

```
rotator.go           # Rotation engine
strategies/
  random.go          # Random selection
  round_robin.go     # Atomic counter-based
  sticky.go          # Session-hash-based
  least_conn.go      # Stats-based selection
  time_based.go      # Interval-based
```

### `internal/credential/` — Credential Management

Resolves credentials from multiple sources.

```
credential.go   # Provider interface
env.go          # Environment variable provider
file.go         # File-based provider
```

### `internal/dashboard/` — Web Dashboard

Embedded HTTP server for live monitoring.

```
server.go       # HTTP server (stdlib net/http)
handlers.go     # API handlers + HTML serving
templates/      # Embedded HTML (go:embed)
```

### `pkg/types/` — Shared Types

```
types.go        # Proxy, Pool, Protocol, RotationStrategy
interfaces.go   # ProxyStore interface for pool/proxy decoupling
```

**`ProxyStore` interface** (added in remediation):
```go
type ProxyStore interface {
    List() []Proxy
    Add(proxy Proxy)
    Remove(id string) bool
    Get(id string) (Proxy, bool)
    Count() int
    UpdateStatus(id string, status string)
}
```

### `internal/health/` — Health Checking

Background health probe with automatic blacklist.

```
checker.go       # Health check engine (background goroutine)
interfaces.go    # ProxyProvider interface for decoupling
tcp.go          # TCP connection probe
```

**`ProxyProvider` interface** (added in remediation):
```go
type ProxyProvider interface {
    List() []types.Proxy
    UpdateStatus(id string, status string)
}
```

### `internal/platform/` — Platform Utilities

Process and proxy server management.

```
process.go        # Process lifecycle (start/stop/restart/reload)
proxy_server.go   # ProxyServer interface + ThreeProxyServer implementation
```

**`ProxyServer` interface** (added in remediation):
```go
type ProxyServer interface {
    Start(ctx context.Context, cfg ProxyServerConfig) error
    Stop(ctx context.Context) error
    IsRunning() bool
    PID() int
    Status() ServerStatus
}
```

### `internal/errs/` — Error Types

Typed sentinel errors for precise error checking.

```
errors.go   # Sentinel errors (ErrNoPrefix, Err3proxyNotFound, etc.)
```

**Sentinel errors** (added in remediation):
```go
var (
    ErrNoPrefix           = errors.New("ipv6_prefix_required")
    ErrInvalidPrefix      = errors.New("invalid_ipv6_prefix")
    Err3proxyNotFound     = errors.New("3proxy_binary_not_found")
    ErrPortInUse          = errors.New("port_already_in_use")
    ErrPoolNotFound       = errors.New("pool_not_found")
    ErrPoolAlreadyRunning = errors.New("pool_already_running")
    ErrPoolExists         = errors.New("pool_already_exists")
    ErrInvalidPort        = errors.New("invalid_port_range")
    ErrPermDenied         = errors.New("permission_denied")
    ErrConfigNotFound     = errors.New("config_file_not_found")
    ErrConfigInvalid      = errors.New("config_invalid")
    ErrCredentialNotFound = errors.New("credential_not_found")
)
```

### `configs/` — Static Assets

```
3proxy.cfg.tmpl   # 3proxy configuration template
```

## Data Flow: Start Command

```
rip start --count 100
    │
    ▼
Load config (Viper) + env overrides
    │
    ▼
Resolve credential (env/file/inline)
    │
    ▼
PoolManager.CreatePool(ctx, PoolOptions{Count: 100})
    │
    ├──► Generator.Batch(100) → []net.IP
    ├──► Users: GenerateUsers(100) → []User
    └──► Pool state → registry.json
    │
    ▼
ProxyManager.Start(ctx, StartOptions{
    Proxies:    pool.Proxies,
    Users:      pool.Users,
    ConfigPath: state.3proxyConfigPath,
})
    │
    ├──► Render 3proxy.cfg from template (100 entries)
    ├──► Write users.conf
    ├──► Spawn 3proxy /path/3proxy.cfg
    ├──► Write PID to state.json
    └──► (Optional) Start health checker goroutine
    │
    ▼
Print: "Started 100 proxies on :10000-10099 (PID: 12345)"
```

## Data Flow: Health Check

```
Background goroutine (health.Checker.Start)
    │
    ▼ (every 5s interval)
For each proxy in pool:
    │
    ├──► Probe.Probe(ctx, ip, port) → ProbeResult
    │         │
    │         ▼
    │    Success?
    │      ├─ YES → MarkHealthy(ip, port)
    │      └─ NO  → failCount++
    │               │
    │               ▼
    │          failCount >= 3?
    │            ├─ YES → Blacklist(ip, port, reason)
    │            └─ NO  → continue
    │
    ▼
Blacklist updated
    │
    ▼
Rotator.NextProxy() → skips blacklisted entries
```

## State Management

| File | Location | Format | Purpose |
|------|----------|--------|---------|
| Config | `./rip.toml` → `~/.config/proxy-ipv6-cli/...` | TOML | User settings |
| State | `~/.local/share/proxy-ipv6-cli/state.json` | JSON | 3proxy PID, runtime |
| Pools | `~/.local/share/proxy-ipv6-cli/pools.json` | JSON | Named pool registry |
| 3proxy.cfg | temp or `~/.local/share/proxy-ipv6-cli/` | text | Generated per pool |
| users.conf | temp or `~/.local/share/proxy-ipv6-cli/` | text | 3proxy users |
| Logs | `~/.local/share/proxy-ipv6-cli/logs/` | text/zip | Access logs + rotation |

## Security Boundaries

```
CLI (user's terminal)
  │  Reads config, env vars
  ▼
rip process (user's privilege)
  │  Manages 3proxy process
  │  Writes state to ~/.local/share/proxy-ipv6-cli/
  ▼
3proxy process (user's privilege)
  │  Binds ports, routes traffic
  ▼
Network (Internet)
```

- Credentials stored as `username:CL:password` in `users.conf` (file permissions: 0600)
- Plain-text credentials in env vars visible to `ps` on some systems — warn users
- Dashboard binds to localhost by default; auth required for non-localhost
- No sensitive data in logs (passwords redacted)

## Error Propagation

```
3proxy error (stderr)
    │
    ▼
proxy.Manager.processOutput() goroutine
    │
    ▼
Structured log line: {level: "error", msg: "...", pool: "default"}
    │
    ▼
CLI command returns exit code 1 + human-readable message
```

## Testing

Unit tests cover critical paths across all packages:

| Package | Test File | Coverage |
|---------|-----------|----------|
| `internal/generator/` | `ipv6_test.go` | IPv6 generation algorithms |
| `internal/health/` | `checker_test.go` | Health check engine |
| `internal/config/` | `generator_test.go`, `config_test.go` | Config loading, 3proxy generation |
| `internal/platform/` | `process_test.go` | Process lifecycle |
| `internal/pool/` | `manager_test.go` | Pool management |
| `internal/proxy/` | `manager_test.go` | Proxy manager |
| `internal/state/` | `state_test.go` | State persistence |

Run tests with:
```bash
make test
make test-coverage  # with coverage report
```

## Extension Points

1. **New proxy backend:** Implement `ProxyServer` interface in `internal/platform/proxy_server.go` (e.g., Dante, Squid)
2. **New rotation strategy:** Implement `Strategy` interface, register in `rotator.go`
3. **New credential provider:** Implement `CredentialProvider` interface, add to `credential.go`
4. **New export format:** Implement `Exporter` interface in `cmd/export.go`
5. **Custom proxy store:** Implement `ProxyStore` interface in `pkg/types/interfaces.go` for alternative backends
