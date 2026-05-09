package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// ProxyServer represents a proxy server process (e.g., 3proxy).
type ProxyServer interface {
	// Start launches the proxy server with the given configuration.
	// Returns error if already running or if binary/config fails.
	Start(ctx context.Context, cfg ProxyServerConfig) error

	// Stop terminates the proxy server gracefully.
	// Sends SIGTERM, waits for grace period, then SIGKILL if needed.
	Stop(ctx context.Context) error

	// IsRunning returns true if the server process is alive.
	IsRunning() bool

	// PID returns the process ID, or 0 if not running.
	PID() int

	// Status returns current server status.
	Status() ServerStatus
}

type ProxyServerConfig struct {
	BinaryPath string
	ConfigPath string
	PIDPath    string
	UsersPath  string
	GracePeriod time.Duration
}

type ServerStatus struct {
	Running   bool
	PID       int
	Uptime    time.Duration
	StartedAt time.Time
	Error     string
}

// ThreeProxyServer is the 3proxy implementation of ProxyServer.
type ThreeProxyServer struct {
	config   ProxyServerConfig
	status   ServerStatus
	pm       *ProcessManager
	mu       sync.RWMutex
}

func NewThreeProxyServer(pm *ProcessManager) *ThreeProxyServer {
	return &ThreeProxyServer{
		pm: pm,
	}
}

func (s *ThreeProxyServer) Start(ctx context.Context, cfg ProxyServerConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status.Running {
		return fmt.Errorf("server already running (PID %d)", s.status.PID)
	}

	s.config = cfg

	if cfg.GracePeriod == 0 {
		cfg.GracePeriod = 5 * time.Second
	}

	proc, err := s.pm.Start(cfg.BinaryPath, cfg.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to start 3proxy: %w", err)
	}

	if cfg.PIDPath != "" {
		if err := EnsureDir(filepath.Dir(cfg.PIDPath)); err != nil {
			_ = proc.Signal(syscall.SIGTERM) // Best-effort cleanup
			return fmt.Errorf("failed to create pid dir: %w", err)
		}
		if err := s.pm.WritePIDFile(cfg.PIDPath, proc.PID); err != nil {
			_ = proc.Signal(syscall.SIGTERM) // Best-effort cleanup
			return fmt.Errorf("failed to write PID file: %w", err)
		}
	}

	s.status = ServerStatus{
		Running:   true,
		PID:       proc.PID,
		StartedAt: time.Now(),
	}
	return nil
}

func (s *ThreeProxyServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.status.Running {
		return nil
	}

	pid := s.status.PID
	grace := s.config.GracePeriod
	if grace == 0 {
		grace = 5 * time.Second
	}

	proc, err := os.FindProcess(pid)
	if err == nil {
		_ = proc.Signal(syscall.SIGTERM) // Best-effort
		time.Sleep(grace)
		if proc.Signal(syscall.Signal(0)) == nil {
			_ = proc.Kill() // Best-effort
		}
	}

	if s.config.PIDPath != "" {
		_ = os.Remove(s.config.PIDPath) // Best-effort cleanup
	}

	s.status.Running = false
	s.status.PID = 0
	return nil
}

func (s *ThreeProxyServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.status.Running {
		return false
	}
	return s.pm.IsRunning(s.status.PID)
}

func (s *ThreeProxyServer) PID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status.PID
}

func (s *ThreeProxyServer) Status() ServerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := s.status
	if status.Running {
		status.Uptime = time.Since(status.StartedAt)
	}
	return status
}
