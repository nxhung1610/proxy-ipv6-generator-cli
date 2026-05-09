# RIP — Architecture

## System Overview

```
User CLI (Kong)
    │
    ▼
Config Loader (TOML, plain)
    │
    ▼
Command Handlers (cmd/cli/*.go)
    │
    ├──► IPv6 Generator (internal/generator/ipv6.go)
    │         │
    │         ▼
    ├──► 3proxy Config Generator (internal/config/generator.go)
    │         │
    │         ▼
    ├──► 3proxy Process Manager (internal/platform/process.go)
    │         │  PID file, start/stop
    │         ▼
    │    3proxy Process (external binary)
    │
    ├──► Pool Manager (internal/pool/manager.go)
    │         │  Multi-pool registry, state.json
    │         ▼
    ├──► Proxy Manager (internal/proxy/manager.go)
    │         │  Proxy list management
    │         ▼
    ├──► Health Checker (internal/health)
    │         │  TCP probe, blacklist
    │         ▼
    └──► State Manager (internal/state/state.go)
              │  JSON persistence with filepath.Clean
              ▼
           State File (JSON)
```

## Components

### `cmd/cli/` — CLI Commands

Kong-based CLI with subcommands. Each subcommand is a separate file.

```
cmd/cli/
├── main.go              # Root CLI, Kong parser, shared context
├── cmd_init.go          # rip init
├── cmd_install.go       # rip install (embedded in main.go)
├── cmd_generate.go      # rip generate
├── cmd_pool.go          # rip pool {start,stop,status,create,remove,list}
├── cmd_config.go        # rip config generate
├── cmd_health.go        # rip health check
├── cmd_export.go        # rip export
└── cmd_version.go       # rip version
```

**Root CLI structure** (main.go):

```go
type CLI struct {
    Init     InitCmd      `cmd:"" help:"Initialize configuration"`
    Install  InstallCmd   `cmd:"install" help:"Install 3proxy (bundled)"`
    Generate GenerateCmd  `cmd:"generate" help:"Generate IPv6 addresses"`
    Pool     PoolCmd      `cmd:"pool" help:"Manage proxy pools"`
    Config   ConfigCmd    `cmd:"config" help:"Configuration management"`
    Health   HealthCmd    `cmd:"health" help:"Health check management"`
    Export   ExportCmd    `cmd:"export" help:"Export proxies"`
    Version  VersionCmd   `cmd:"version" help:"Show version"`
}
```

---

### `internal/generator/` — IPv6 Address Generation

Generates IPv6 addresses from a prefix using random IID (Interface Identifier) generation.

```
internal/generator/
└── ipv6.go      # IPv6 address generation with math/big.Int counter
```

**Key functions:**

```go
func New(prefix string, maxAddresses, count int) (*Generator, error)
func (g *Generator) Generate(count int) ([]types.Proxy, error)
```

---

### `internal/proxy/` — Proxy Management

Manages the proxy list.

```
internal/proxy/
└── manager.go    # Proxy list management (Add, Remove, List, UpdateStatus)
```

---

### `internal/pool/` — Pool Management

Multi-pool registry with state persistence.

```
internal/pool/
└── manager.go    # Pool CRUD, start/stop, proxy management
```

**Pool state includes:** ID, Name, Prefix, Ports, Protocol, Count, Status, Proxies

---

### `internal/health/` — Health Checking

TCP-based health probing with failure tracking.

```
internal/health/
├── checker.go       # Health check engine
├── tcp.go          # TCP connection probe
└── interfaces.go   # ProxyProvider interface
```

**ProxyProvider interface:**

```go
type ProxyProvider interface {
    List() []types.Proxy
    UpdateStatus(id string, status string)
}
```

---

### `internal/config/` — Configuration

Viper-based TOML config loading and 3proxy config generation.

```
internal/config/
├── config.go      # TOML config loading via Viper (env var overrides, auto-discovery)
└── generator.go   # 3proxy.cfg generation via text/template
```

---

### `internal/state/` — State Management

Simple JSON state persistence with security hardening.

```
internal/state/
└── state.go       # JSON state with filepath.Clean, 0600 permissions
```

---

### `internal/platform/` — Platform Utilities

Process and proxy server management.

```
internal/platform/
├── process.go        # Process lifecycle (start/stop, PID files)
└── proxy_server.go   # ProxyServer interface + ThreeProxyServer
```

**ProxyServer interface:**

```go
type ProxyServer interface {
    Start(ctx context.Context, cfg ProxyServerConfig) error
    Stop(ctx context.Context) error
    IsRunning() bool
    PID() int
    Status() ServerStatus
}
```

---

### `internal/errs/` — Error Types

Typed sentinel errors.

```
internal/errs/
└── errors.go       # Sentinel errors (ErrNoPrefix, Err3proxyNotFound, etc.)
```

---

### `pkg/types/` — Shared Types

```
pkg/types/
├── types.go        # Proxy, Pool, Protocol, RotationStrategy
└── interfaces.go   # ProxyStore interface
```

**ProxyStore interface:**

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

---

## Data Flow: Pool Start

```
rip pool start
    │
    ▼
PoolManager.GetByName("default")
    │
    ▼
Generate 3proxy config (config.Generate3proxyConfig)
    │
    ▼
Write config to ~/.local/share/proxy-ipv6-cli/pools/<pool-id>/3proxy.cfg
    │
    ▼
Detect 3proxy binary (platform.Detect3proxy)
    │
    ▼
Start 3proxy process (platform.ProcessManager.Start)
    │
    ▼
Write PID file
    │
    ▼
Print: "Pool 'default' started (PID 12345, 100 proxies)"
```

---

## State Management

| File | Location | Format | Purpose |
|------|----------|--------|---------|
| Config | `~/.config/proxy-ipv6-cli/config.toml` | TOML | User settings |
| State | `~/.local/share/proxy-ipv6-cli/state.json` | JSON | Runtime state |
| Pools | In state.json | JSON | Named pool registry |
| 3proxy.cfg | `~/.local/share/proxy-ipv6-cli/pools/<pool-id>/` | text | Generated per pool |
| PID files | `~/.local/share/proxy-ipv6-cli/pools/<pool-id>/` | text | Process ID tracking |

---

## Security

- Credentials stored in 3proxy config with 0600 permissions
- `filepath.Clean` used for all file paths to prevent traversal
- Password omitted from JSON export
- No sensitive data in process output

---

## Testing

Unit tests cover critical paths:

| Package | Test File | Coverage |
|---------|-----------|----------|
| `internal/generator/` | `ipv6_test.go` | IPv6 generation |
| `internal/health/` | `checker_test.go` | Health check engine |
| `internal/config/` | `generator_test.go`, `config_test.go` | Config loading, 3proxy generation |
| `internal/platform/` | `process_test.go` | Process lifecycle |
| `internal/pool/` | `manager_test.go` | Pool management |
| `internal/state/` | `state_test.go` | State persistence |

Run tests with:

```bash
go test ./...
make test-coverage  # with coverage report
```

---

## Extension Points

1. **New proxy backend:** Implement `ProxyServer` interface in `internal/platform/proxy_server.go`
2. **New export format:** Extend `cmd_export.go` switch statement
3. **Custom proxy store:** Implement `ProxyStore` interface in `pkg/types/interfaces.go`
