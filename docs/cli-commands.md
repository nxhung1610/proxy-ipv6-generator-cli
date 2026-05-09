# CLI Commands Reference

Complete reference for all `rip` commands, flags, and examples.

---

## Global Flags

These flags are available on every command:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | auto-discovered | Path to config file |
| `--log-level` | | `info` | `debug`, `info`, `warn`, `error` |
| `--log-format` | | `text` | `text` or `json` |
| `--verbose` | `-v` | | Enable debug output |
| `--dry-run` | | | Preview without making changes |
| `--yes` | `-y` | | Skip confirmation prompts |
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

**Behavior:**
- Creates `~/.config/proxy-ipv6-cli/config.toml` by default
- If a config file already exists, does nothing
- Sets reasonable defaults for all configuration values

**Examples:**

```bash
# Create default config at ~/.config/proxy-ipv6-cli/config.toml
rip init

# Create config in current directory
rip init ./rip.toml
```

**Note:** The interactive prompts and flags (`--prefix`, `--port`, `--username`, etc.) shown in documentation are planned features not yet implemented. Currently, edit the config file manually after initialization.

---

## `start` — Start Proxy Pool

Start the proxy server pool.

```bash
rip start [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--count` | config `count` | Number of proxy instances to start |
| `--port` | config `basePort` | Starting port number |
| `--pool` | `default` | Pool name to start |
| `--strategy` | `random` | Initial rotation strategy |
| `--no-health` | false | Disable health checking |
| `--detach` | false | Run in background, write PID |

**Examples:**

```bash
# Start 100 proxies from config
rip start

# Start 500 proxies on ports 20000-20499
rip start --count 500 --port 20000

# Preview without starting
rip start --count 100 --dry-run

# Start specific pool
rip start --pool us-east --count 50
```

**Output:**

```
Started proxy pool "default"
  Prefix:   2001:db8::/64
  Proxies:  100 (ports 10000-10099)
  Protocol: socks5
  PID:      12345
  Logs:     ~/.local/share/proxy-ipv6-cli/logs/

Credentials:
  Username: myuser
  Password: mypass

Test:
  curl --proxy socks5://myuser:mypass@localhost:10000 https://ifconfig.me
```

**Exit codes:** 0 (started), 1 (already running), 2 (config error), 3 (3proxy not found), 4 (port in use)

---

## `stop` — Stop Proxy Pool

Stop the running proxy server.

```bash
rip stop [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--pool` | `default` | Pool name to stop |
| `--force` | false | Use SIGKILL instead of SIGTERM |
| `--timeout` | `10s` | Wait time before SIGKILL |

**Examples:**

```bash
rip stop
rip stop --pool us-east
rip stop --force    # Hard kill
```

**Output:**

```
Stopped proxy pool "default" (was PID 12345)
```

**Exit codes:** 0 (stopped), 5 (pool not found/running)

---

## `status` — Show Status

Show the current status of proxy pools.

```bash
rip status [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--pool` | all pools | Pool name to check |
| `--json` | false | JSON output |

**Examples:**

```bash
rip status
rip status --pool default --json
```

**Output (text):**

```
● rip: running (PID 12345)
  Pool:       default
  Prefix:     2001:db8::/64
  Proxies:    100 (ports 10000-10099)
  Protocol:   socks5
  Strategy:   random
  Healthy:    97/100
  Unhealthy:  3 (blacklisted)
  Started:    2026-05-09T19:50:00+07:00
  Uptime:     2h 34m
```

**Output (JSON):**

```json
{
  "pool": "default",
  "status": "running",
  "pid": 12345,
  "prefix": "2001:db8::/64",
  "proxies": 100,
  "portRange": "10000-10099",
  "protocol": "socks5",
  "healthy": 97,
  "unhealthy": 3,
  "startedAt": "2026-05-09T19:50:00+07:00"
}
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

**Note:** The `--strategy`, `--start`, `--end`, `--seed`, and `--format` flags are documented in the design spec but not yet implemented.

**Examples:**

```bash
# Generate 100 addresses (default)
rip generate --count 100

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
    "username": "user",
    "password": "pass",
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
rip pool create [flags]
```

**Flags:** `--name`, `--prefix`, `--count`, `--port`, `--protocol`, `--credential`

**Examples:**

```bash
rip pool create \
  --name us-east \
  --prefix 2001:db8:1::/64 \
  --count 50 \
  --port 10000 \
  --protocol socks5 \
  --credential env:MY_US_EAST_CREDS

rip pool create --name default --prefix 2001:db8::/64 --count 100
```

### `pool list`

List all pools.

```bash
rip pool list
```

**Output:**

```
NAME       PREFIX              PROXIES  PORT       STATUS    CREATED
default    2001:db8::/64       100      10000-10099  running  2026-05-09
us-east    2001:db8:1::/64     50       10000-10049  stopped  2026-05-09
```

### `pool remove`

Remove a pool.

```bash
rip pool remove <name>
```

```bash
rip pool remove us-east
# ? Remove pool "us-east" (50 proxies)? This cannot be undone.
#   [Y/n] y
# Removed pool "us-east"
```

### `pool scale`

Scale a pool's proxy count.

```bash
rip pool scale <name> --count <n>
```

```bash
rip pool scale default --count 500
# Scaling pool "default": 100 → 500 proxies
# Adding ports 10100-10499
# Reloading 3proxy configuration...
# Scaled successfully.
```

---

## `config` — Configuration Management

### `config set`

Set a configuration value.

```bash
rip config set <key> <value>
```

```bash
rip config set server.prefix "2001:db8::/64"
rip config set server.count 500
rip config set logging.level "debug"
```

### `config get`

Get a configuration value.

```bash
rip config get <key>
```

```bash
rip config get server.prefix
# 2001:db8::/64
```

### `config list`

List all configuration values.

```bash
rip config list
```

**Output:**

```
server.prefix      = "2001:db8::/64"
server.basePort   = 10000
server.protocol   = "socks5"
server.count      = 100
auth.credentialRef = "env:MY_PROXY_USER"
logging.level     = "info"
logging.format    = "text"
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
| `--pool` | `default` | Pool name |
| `--format` | `txt` | `txt`, `json`, `csv` |
| `--output` | stdout | Output file path |
| `--include-status` | false | Include health status (JSON only) |

**Formats:**

**`txt` (default):**
```
2001:db8::1:10000:user:pass
2001:db8::2:10001:user:pass
```

**`json`:**
```json
[
  {"address": "2001:db8::1", "port": 10000, "username": "user", "password": "pass"},
  {"address": "2001:db8::2", "port": 10001, "username": "user", "password": "pass"}
]
```

**`csv`:**
```csv
id,address,port,protocol,status
<uuid>,2001:db8::1,10000,socks5,healthy
<uuid>,2001:db8::2,10001,socks5,healthy
```

**Examples:**

```bash
rip export --format txt
rip export --format json --output proxies.json
rip export --pool us-east --format csv --output us-east-proxies.csv
```

---

## `rotate` — Rotate IPs

Manually trigger IP rotation.

```bash
rip rotate [flags]
```

**Flags:** `--pool` (default: `default`), `--strategy` (override)

```bash
# Rotate with default strategy
rip rotate

# Rotate specific pool with round-robin
rip rotate --pool us-east --strategy round-robin

# Force rotate unhealthy proxies
rip rotate --pool default --strategy random
```

---

## `health` — Health Check Management

### `health check`

Run a one-time health check on all proxies.

```bash
rip health check
```

**Output:**

```
Checking 100 proxies...
● 2001:db8::1:10000    healthy      12ms
● 2001:db8::2:10001    healthy       8ms
○ 2001:db8::3:10002    unhealthy    timeout (3/3 failed)
● 2001:db8::4:10003    healthy      15ms
...

Healthy:  97/100
Unhealthy: 3 (blacklisted)
Duration: 2.4s
```

### `health list`

List health status of all proxies.

```bash
rip health list
```

### `health clear`

Remove a proxy from the blacklist.

```bash
rip health clear 2001:db8::3:10002
```

---

## `logs` — Log Management

### `logs tail`

Stream live log entries.

```bash
rip logs tail
```

### `logs stats`

Show aggregate statistics.

```bash
rip logs stats [flags]
```

**Flags:** `--since` (e.g. `1h`, `24h`, `7d`)

**Output:**

```
Proxy Pool Stats (last 1 hour)
────────────────────────────────
Total requests:  12,450
Bandwidth in:    1.2 GB
Bandwidth out:   8.7 GB
Active users:    3
Errors:          12 (0.1%)

Top prefixes:
  2001:db8:1:4::/64    8,200 requests
  2001:db8:1:5::/64    4,250 requests

Top destination ports:
  443  11,200
  80     1,250
```

### `logs export`

Export logs to a file.

```bash
rip logs export --format json --since 24h --output logs.json
```

---

## `dashboard` — Web Dashboard

Start the embedded web dashboard.

```bash
rip dashboard [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 8080 | Dashboard HTTP port |
| `--bind` | `localhost` | Bind address |
| `--auth` | none | Basic auth (`user:pass`) |
| `--detach` | false | Run in background |

**Examples:**

```bash
# Start dashboard on localhost:8080
rip dashboard

# Start on all interfaces (with auth)
rip dashboard --bind 0.0.0.0 --port 8080 --auth admin:secret123

# Run in background
rip dashboard --detach
```

**Dashboard shows:**
- Pool status (healthy/unhealthy count)
- Live request rate
- Top IPv6 addresses by usage
- Proxy health table with latency, fail count
- Auto-refresh every 5 seconds

---

## `version` — Version Info

```bash
rip version
```

**Output:**

```
rip v0.1.0
  commit:  abc1234
  date:    2026-05-09
  built:   darwin/arm64
```

---

## `completions` — Shell Completions

Generate shell completion scripts.

```bash
# Bash
rip completions bash > /usr/local/etc/bash_completion.d/rip

# Zsh
rip completions zsh > "${fpath[1]}/_rip"

# Fish
rip completions fish > ~/.config/fish/completions/rip.fish

# PowerShell
rip completions powershell >> $PROFILE
```

---

## Environment Variables

All config keys can be overridden via environment variables:

| Config Key | Environment Variable |
|-----------|---------------------|
| `server.prefix` | `PROXY_IPV6_SERVER_PREFIX` |
| `server.basePort` | `PROXY_IPV6_SERVER_BASE_PORT` |
| `server.protocol` | `PROXY_IPV6_SERVER_PROTOCOL` |
| `auth.credentialRef` | `PROXY_IPV6_AUTH_CREDENTIAL_REF` |
| `logging.level` | `PROXY_IPV6_LOGGING_LEVEL` |
| `proxy.binaryPath` | `PROXY_IPV6_PROXY_BINARY_PATH` |

**Credential env var format:** `PROXY_IPV6_AUTH_USERNAME` and `PROXY_IPV6_AUTH_PASSWORD` (when using `env:` ref).
