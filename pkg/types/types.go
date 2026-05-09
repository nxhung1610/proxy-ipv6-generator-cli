package types

type Protocol string

const (
	ProtocolHTTP   Protocol = "http"
	ProtocolSOCKS5 Protocol = "socks5"
)

type RotationStrategy string

const (
	StrategyRandom     RotationStrategy = "random"
	StrategyRoundRobin RotationStrategy = "round-robin"
	StrategySticky    RotationStrategy = "sticky"
)

type ProxyStatus string

const (
	ProxyStatusHealthy   ProxyStatus = "healthy"
	ProxyStatusUnhealthy ProxyStatus = "unhealthy"
	ProxyStatusUnknown   ProxyStatus = "unknown"
)

type PoolStatus string

const (
	PoolStatusRunning PoolStatus = "running"
	PoolStatusStopped PoolStatus = "stopped"
	PoolStatusError   PoolStatus = "error"
)

type Proxy struct {
	ID       string   `json:"id"`
	Address  string   `json:"address"`
	Port     int      `json:"port"`
	Protocol Protocol `json:"protocol"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Status   string   `json:"status"`
}

type Pool struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Prefix         string   `json:"prefix"`
	BasePort       int      `json:"basePort"`
	Count          int      `json:"count"`
	Protocol       Protocol `json:"protocol"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"createdAt"`
	CredentialRef  string   `json:"credentialRef"`
}

type HealthResult struct {
	ProxyID   string `json:"proxyId"`
	Address   string `json:"address"`
	Port      int    `json:"port"`
	Healthy   bool   `json:"healthy"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
}

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Auth     AuthConfig     `toml:"auth"`
	Proxy    ProxyConfig    `toml:"proxy"`
	Logging  LoggingConfig  `toml:"logging"`
	Health   HealthConfig   `toml:"health"`
	Pools    []PoolConfig   `toml:"pools"`
}

type ServerConfig struct {
	Prefix   string `toml:"prefix"`
	BasePort int    `toml:"basePort"`
	Protocol string `toml:"protocol"`
	Count    int    `toml:"count"`
}

type AuthConfig struct {
	CredentialRef string `toml:"credentialRef"`
}

type ProxyConfig struct {
	BinaryPath       string `toml:"binaryPath"`
	MaxConn         int    `toml:"maxConn"`
	BandwidthLimitKBps int `toml:"bandwidthLimitKBps"`
}

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

type HealthConfig struct {
	Enabled         bool `toml:"enabled"`
	IntervalSeconds int `toml:"intervalSeconds"`
	TimeoutSeconds  int `toml:"timeoutSeconds"`
	MaxFailures    int  `toml:"maxFailures"`
}

type PoolConfig struct {
	Name   string `toml:"name"`
	Prefix string `toml:"prefix"`
	Ports  string `toml:"ports"`
}

type State struct {
	Version string          `json:"version"`
	Pools   []Pool         `json:"pools"`
	Proxy3  ThreeProxyState `json:"3proxy"`
}

type ThreeProxyState struct {
	PID        int    `json:"pid"`
	ConfigPath string `json:"configPath"`
	StartedAt  string `json:"startedAt"`
}
