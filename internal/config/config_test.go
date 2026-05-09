package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_NoConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "does-not-exist.toml")

	// No config file exists — Load should return defaults
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed with no config: %v", err)
	}

	if cfg.Server.BasePort != 10000 {
		t.Errorf("expected default BasePort=10000, got %d", cfg.Server.BasePort)
	}
	if cfg.Server.Protocol != "socks5" {
		t.Errorf("expected default Protocol=socks5, got %q", cfg.Server.Protocol)
	}
	if cfg.Health.IntervalSeconds != 5 {
		t.Errorf("expected default IntervalSeconds=5, got %d", cfg.Health.IntervalSeconds)
	}
	if cfg.Health.TimeoutSeconds != 3 {
		t.Errorf("expected default TimeoutSeconds=3, got %d", cfg.Health.TimeoutSeconds)
	}
	if cfg.Health.MaxFailures != 3 {
		t.Errorf("expected default MaxFailures=3, got %d", cfg.Health.MaxFailures)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	content := `
[server]
prefix = "2001:db8::/64"
basePort = 20000
protocol = "http"
count = 50

[health]
enabled = false
intervalSeconds = 10
timeoutSeconds = 5
maxFailures = 5

[proxy]
maxConn = 1000
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Prefix != "2001:db8::/64" {
		t.Errorf("expected prefix %q, got %q", "2001:db8::/64", cfg.Server.Prefix)
	}
	if cfg.Server.BasePort != 20000 {
		t.Errorf("expected BasePort=20000, got %d", cfg.Server.BasePort)
	}
	if cfg.Server.Protocol != "http" {
		t.Errorf("expected Protocol=http, got %q", cfg.Server.Protocol)
	}
	if cfg.Server.Count != 50 {
		t.Errorf("expected Count=50, got %d", cfg.Server.Count)
	}
	if cfg.Health.IntervalSeconds != 10 {
		t.Errorf("expected IntervalSeconds=10, got %d", cfg.Health.IntervalSeconds)
	}
	if cfg.Health.MaxFailures != 5 {
		t.Errorf("expected MaxFailures=5, got %d", cfg.Health.MaxFailures)
	}
	if cfg.Proxy.MaxConn != 1000 {
		t.Errorf("expected MaxConn=1000, got %d", cfg.Proxy.MaxConn)
	}
}

func TestInitConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	err := InitConfig(cfgPath)
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// File should exist
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}

	// Should be idempotent — calling again should not error
	if err := InitConfig(cfgPath); err != nil {
		t.Errorf("InitConfig should be idempotent: %v", err)
	}

	// Verify contents
	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if content == "" {
		t.Error("config file should not be empty")
	}
	if !strings.Contains(content, "server.prefix") {
		t.Error("config file missing server.prefix")
	}
}
