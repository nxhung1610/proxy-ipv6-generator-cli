package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
	"github.com/spf13/viper"
)

func Load(configPath string) (*types.Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("toml")

	homeDir, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(homeDir, ".config", "proxy-ipv6-cli"))
		v.AddConfigPath(filepath.Join(homeDir, ".config", "proxy-ipv6-generator-cli"))
	}

	v.AddConfigPath(".")
	v.AddConfigPath("~/.config/proxy-ipv6-cli")

	if configPath != "" {
		v.SetConfigFile(configPath)
	}

	v.SetEnvPrefix("PROXY_IPV6")
	v.AutomaticEnv()

	v.SetDefault("server.basePort", 10000)
	v.SetDefault("server.protocol", "socks5")
	v.SetDefault("server.count", 100)
	v.SetDefault("proxy.maxConn", 500)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("health.enabled", true)
	v.SetDefault("health.intervalSeconds", 5)
	v.SetDefault("health.timeoutSeconds", 3)
	v.SetDefault("health.maxFailures", 3)

	if err := v.ReadInConfig(); err != nil {
		// Allow missing file when a specific path is given — return defaults.
		if _, ok := err.(viper.ConfigFileNotFoundError); ok || errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg types.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func InitConfig(configPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	dir := filepath.Join(homeDir, ".config", "proxy-ipv6-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	defaultConfig := `server.prefix = ""
server.basePort = 10000
server.protocol = "socks5"
server.count = 100

[auth]
credentialRef = ""

[proxy]
binaryPath = ""
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
`

	cfgFile := filepath.Join(dir, "config.toml")
	if configPath != "" {
		cfgFile = configPath
	}

	if _, err := os.Stat(cfgFile); err == nil {
		return nil
	}

	if err := os.WriteFile(cfgFile, []byte(defaultConfig), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
