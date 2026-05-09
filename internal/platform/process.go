package platform

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

type ProcessManager struct{}

func NewProcessManager() *ProcessManager {
	return &ProcessManager{}
}

func (pm *ProcessManager) Start(path string, args ...string) (*Process, error) {
	if path == "" {
		path, _ = pm.Detect3proxy()
		if path == "" {
			return nil, fmt.Errorf("3proxy binary not found; run 'rip install' or install 3proxy manually")
		}
	}

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	return &Process{
		PID:   cmd.Process.Pid,
		cmd:   cmd,
	}, nil
}

func (pm *ProcessManager) Stop(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := proc.Kill(); err != nil {
		if sysErr, ok := err.(*os.SyscallError); ok {
			if sysErr.Err == syscall.ESRCH {
				return nil
			}
		}
		return fmt.Errorf("failed to kill process: %w", err)
	}

	proc.Wait()
	return nil
}

func (pm *ProcessManager) IsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func (pm *ProcessManager) FindBinary(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// Detect3proxy searches for 3proxy in this order:
//  1. Bundled vendor binary (./vendor/3proxy/3proxy-<os>-<arch>) relative to the running binary
//  2. User-local install (~/.local/share/rip/bin/3proxy or %LOCALAPPDATA%\rip\bin\3proxy.exe)
//  3. System paths (/usr/local/bin/3proxy, /usr/bin/3proxy, /opt/3proxy/3proxy)
//  4. $PATH
//
// When a binary is found outside system paths, VerifyBinary logs a warning.
func (pm *ProcessManager) Detect3proxy() (string, error) {
	if bin, err := pm.vendorBinary(); err == nil && bin != "" {
		return bin, nil
	}

	if bin := pm.localBinary(); bin != "" {
		if _, err := os.Stat(bin); err == nil {
			VerifyBinary(bin)
			return bin, nil
		}
	}

	paths := []string{
		"/usr/local/bin/3proxy",
		"/usr/bin/3proxy",
		"/opt/3proxy/3proxy",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	if path, err := exec.LookPath("3proxy"); err == nil {
		VerifyBinary(path)
		return path, nil
	}

	return "", fmt.Errorf("3proxy binary not found; run 'rip install' or install 3proxy manually")
}

// systemBinaryPaths lists trusted system directories for 3proxy binaries.
var systemBinaryPaths = []string{"/usr/bin", "/usr/local/bin", "/opt/bin"}

// VerifyBinary checks whether the given binary path is in a trusted system location
// and logs a warning if it is not. This helps detect potentially compromised binaries
// in user-writable locations like ~/.local/share/rip/bin/.
func VerifyBinary(path string) {
	if path == "" {
		return
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return
	}
	for _, sp := range systemBinaryPaths {
		if strings.HasPrefix(absPath, sp) {
			return
		}
	}
	log.Printf("warning: 3proxy binary '%s' is not in a system path. "+
		"Ensure you trust this binary.", path)
}

// vendorBinary returns the bundled 3proxy binary path relative to the running binary.
// On macOS/Darwin there is no official 3proxy binary, so this returns "" (caller should
// fall back to local install or PATH).
func (pm *ProcessManager) vendorBinary() (string, error) {
	// 3proxy has no official macOS binary; skip vendor lookup on Darwin.
	if runtime.GOOS == "darwin" {
		return "", fmt.Errorf("not available on darwin")
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)
	vendorPath := filepath.Join(exeDir, "3proxy-bin", BinaryName())
	if _, err := os.Stat(vendorPath); err == nil {
		return vendorPath, nil
	}
	return "", fmt.Errorf("not found in vendor")
}

// localBinary returns the user-local 3proxy binary path.
func (pm *ProcessManager) localBinary() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", "rip", "bin", "3proxy.exe")
	}
	return filepath.Join(home, ".local", "share", "rip", "bin", BinaryName())
}

// InstallPath is the directory where `rip install` installs 3proxy binaries.
func (pm *ProcessManager) InstallPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", "rip", "bin")
	}
	return filepath.Join(home, ".local", "share", "rip", "bin")
}

// ConfigPath returns the 3proxy config directory for a given pool.
func (pm *ProcessManager) ConfigPath(poolID string) string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", "rip", "pools", poolID)
	}
	return filepath.Join(home, ".local", "share", "rip", "pools", poolID)
}

// PIDPath returns the PID file path for a given pool.
func (pm *ProcessManager) PIDPath(poolID string) string {
	return filepath.Join(pm.ConfigPath(poolID), "3proxy.pid")
}

// EnsureDir creates the directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0700)
}

// BinaryName returns the platform-specific 3proxy binary name.
func BinaryName() string {
	if runtime.GOOS == "windows" {
		return "3proxy.exe"
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		return "3proxy-" + runtime.GOOS + "-" + runtime.GOARCH
	}
	return "3proxy"
}

func (pm *ProcessManager) GetPIDFromFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

func (pm *ProcessManager) WritePIDFile(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0600)
}

type Process struct {
	PID int
	cmd *exec.Cmd
}

func (p *Process) Wait() error {
	return p.cmd.Wait()
}

func (p *Process) Signal(sig syscall.Signal) error {
	return p.cmd.Process.Signal(sig)
}
