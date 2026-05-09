# CLI Commands Reference

Complete reference for all `rip` commands, flags, and examples.

---

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | auto-discovered | Path to config file |
| `--help` | `-h` | | Show help |

---

## `init` — Initialize Configuration

Create a new configuration file with default values.

```bash
rip init [config-path]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `config-path` | Optional path for the config file (default: `~/.config/proxy-ipv6-cli/config.toml`) |

**Examples:**

```bash
# Create default config at ~/.config/proxy-ipv6-cli/config.toml
rip init

# Create config in current directory
rip init ./rip.toml
```

---

## `install` — Install 3proxy

Install 3proxy binary for the current host platform.

```bash
rip install
```

**Behavior:**
- On Linux/Windows: copies bundled `3proxy-bin/3proxy-<os>-<arch>` next to the executable
- On macOS: copies system-installed 3proxy if found in PATH
- Falls back to copying from user-local install path
- If no binary is available, prints instructions for compiling from source

**Examples:**

```bash
rip install
# 3proxy installed to ~/.local/share/proxy-ipv6-cli/bin/3proxy
```

---

## `generate` — Generate IPv6 Addresses

Generate IPv6 addresses from the configured prefix.

```bash
rip generate [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--count` / `-n` | config count or 100 | Number of addresses to generate |
| `--prefix` / `-p` | config prefix | IPv6 prefix (e.g., 2001:db8::/64) |
| `--output` / `-o` | stdout | Output file path |

**Examples:**

```bash
# Generate 100 addresses (default)
rip generate

# Generate with custom prefix
rip generate --prefix 2001:db8:1::/48 --count 50

# Output to file
rip generate --count 100 --output addresses.json
```

**Output (JSON):**

```json
[
  {
    "id": "uuid",
    "address": "2001:db8::1",
    "port": 10000,
    "protocol": "socks5",
    "status": "unknown"
  }
]
```

---

## `pool` — Pool Management

Manage multiple named proxy pools.

### `pool create`

Create a new named pool.

```bash
rip pool create <name> <prefix> <ports> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `name` | Pool name |
| `prefix` | IPv6 prefix (e.g., 2001:db8::/64) |
| `ports` | Port range (e.g., 10000-10100) |

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--protocol` / `-p` | `socks5` | Protocol: `socks5` or `http` |

**Examples:**

```bash
rip pool create default 2001:db8::/64 10000-10099

rip pool create us-east 2001:db8:1::/64 20000-20049 --protocol http
```

---

### `pool start`

Start a pool.

```bash
rip pool start [pool-name]
```

**Arguments:**

| Argument | Default | Description |
|----------|---------|-------------|
| `pool-name` | `default` | Pool name to start |

**Behavior:**
- Reads pool configuration
- Generates 3proxy config file
- Starts 3proxy process
- Writes PID file

**Examples:**

```bash
rip pool start
rip pool start default
rip pool start us-east
```

**Output:**

```
Pool 'default' started (PID 12345, 100 proxies)
```

---

### `pool stop`

Stop a running pool.

```bash
rip pool stop [pool-name]
```

**Arguments:**

| Argument | Default | Description |
|----------|---------|-------------|
| `pool-name` | `default` | Pool name to stop |

**Behavior:**
- Stops the 3proxy process
- Removes PID file
- Updates pool status

**Examples:**

```bash
rip pool stop
rip pool stop default
```

**Output:**

```
Pool 'default' stopped
```

---

### `pool status`

Show pool status.

```bash
rip pool status [pool-name]
```

**Arguments:**

| Argument | Default | Description |
|----------|---------|-------------|
| `pool-name` | first pool | Pool name to check |

**Output (JSON):**

```json
{
  "pool": "default",
  "status": "running",
  "prefix": "2001:db8::/64",
  "proxies": 100,
  "protocol": "socks5"
}
```

**Examples:**

```bash
rip pool status
rip pool status default
```

---

### `pool list`

List all pools.

```bash
rip pool list
```

**Output (JSON):**

```json
[
  {
    "id": "pool-uuid",
    "name": "default",
    "prefix": "2001:db8::/64",
    "ports": "10000-10099",
    "protocol": "socks5",
    "count": 100,
    "status": "running"
  }
]
```

**Examples:**

```bash
rip pool list
```

---

### `pool remove`

Remove a pool.

```bash
rip pool remove <pool-name> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `pool-name` | Pool name to remove |

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` / `-y` | false | Skip confirmation prompt |

**Examples:**

```bash
# Interactive removal
rip pool remove us-east

# Skip confirmation
rip pool remove us-east --yes
```

---

## `config` — Configuration Management

### `config generate`

Generate 3proxy configuration for a pool.

```bash
rip config generate [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--pool` / `-p` | `default` | Pool name |
| `--output` / `-o` | stdout | Output file path |

**Examples:**

```bash
# Print to stdout
rip config generate

# Generate for specific pool
rip config generate --pool us-east

# Save to file
rip config generate --output /etc/3proxy/proxy.cfg
```

---

## `health` — Health Check Management

### `health check`

Run a one-time health check on proxies.

```bash
rip health check [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--pool` / `-p` | all proxies | Pool name to check |

**Behavior:**
- Uses configured timeout and interval settings
- Runs for up to 30 seconds
- Returns JSON results

**Output (JSON):**

```json
[
  {
    "proxyId": "uuid",
    "address": "2001:db8::1",
    "port": 10000,
    "status": "healthy",
    "latencyMs": 12
  }
]
```

**Examples:**

```bash
rip health check
rip health check --pool default
```

---

## `export` — Export Proxies

Export proxy list in various formats.

```bash
rip export [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format` / `-f` | `json` | Output format: `json`, `txt`, `csv` |
| `--output` / `-o` | stdout | Output file path |

**Formats:**

**`json` (default):**

```json
[
  {"address": "2001:db8::1", "port": 10000, "protocol": "socks5", "status": "healthy"}
]
```

**`txt`:**

```
2001:db8::1:10000
2001:db8::2:10001
```

**`csv`:**

```csv
id,address,port,protocol,status
uuid1,2001:db8::1,10000,socks5,healthy
uuid2,2001:db8::2,10001,socks5,healthy
```

**Examples:**

```bash
rip export --format txt
rip export --format json --output proxies.json
rip export --format csv --output proxies.csv
```

---

## `version` — Version Info

Show version information.

```bash
rip version
```

**Output:**

```
rip version v0.1.0
```

---

## Environment Variables

All config keys can be overridden via environment variables using the `PROXY_IPV6_` prefix:

| Config Key | Environment Variable |
|-----------|---------------------|
| `server.prefix` | `PROXY_IPV6_SERVER_PREFIX` |
| `server.basePort` | `PROXY_IPV6_SERVER_BASE_PORT` |
| `server.protocol` | `PROXY_IPV6_SERVER_PROTOCOL` |
| `server.count` | `PROXY_IPV6_SERVER_COUNT` |
| `auth.credentialRef` | `PROXY_IPV6_AUTH_CREDENTIAL_REF` |
| `proxy.binaryPath` | `PROXY_IPV6_PROXY_BINARY_PATH` |
| `proxy.maxConn` | `PROXY_IPV6_PROXY_MAX_CONN` |
| `proxy.bandwidthLimitKBps` | `PROXY_IPV6_PROXY_BANDWIDTH_LIMIT_KBPS` |
| `logging.level` | `PROXY_IPV6_LOGGING_LEVEL` |
| `logging.format` | `PROXY_IPV6_LOGGING_FORMAT` |
| `health.enabled` | `PROXY_IPV6_HEALTH_ENABLED` |
| `health.intervalSeconds` | `PROXY_IPV6_HEALTH_INTERVAL_SECONDS` |
| `health.timeoutSeconds` | `PROXY_IPV6_HEALTH_TIMEOUT_SECONDS` |
| `health.maxFailures` | `PROXY_IPV6_HEALTH_MAX_FAILURES` |

---

## See Also

- [Configuration](./configuration.md)
- [Architecture](./architecture.md)
