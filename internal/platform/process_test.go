package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestProcessManager_NewProcessManager(t *testing.T) {
	pm := NewProcessManager()
	if pm == nil {
		t.Error("NewProcessManager returned nil")
	}
}

func TestIsRunning_InvalidPID(t *testing.T) {
	pm := NewProcessManager()

	if pm.IsRunning(-1) {
		t.Error("PID -1 should not be running")
	}
	if pm.IsRunning(0) {
		t.Error("PID 0 should not be running")
	}
	if pm.IsRunning(999999999) {
		t.Error("non-existent PID should not be running")
	}
}

func TestIsRunning_ValidPID(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("test only valid on Unix-like systems")
	}

	pm := NewProcessManager()
	if !pm.IsRunning(1) {
		t.Skip("PID 1 not available in this environment")
	}
}

func TestWritePIDFile(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "test.pid")

	pm := NewProcessManager()
	if err := pm.WritePIDFile(pidPath, 12345); err != nil {
		t.Fatalf("WritePIDFile failed: %v", err)
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("cannot read PID file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "12345" {
		t.Errorf("expected '12345', got: %s", string(data))
	}
}

func TestReadPIDFile(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "test.pid")

	os.WriteFile(pidPath, []byte("54321"), 0600)

	pm := NewProcessManager()
	pid, err := pm.GetPIDFromFile(pidPath)
	if err != nil {
		t.Fatalf("GetPIDFromFile failed: %v", err)
	}
	if pid != 54321 {
		t.Errorf("expected 54321, got %d", pid)
	}
}

func TestReadPIDFile_NotFound(t *testing.T) {
	pm := NewProcessManager()
	_, err := pm.GetPIDFromFile("/nonexistent/path/to/file.pid")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestReadPIDFile_InvalidContent(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "test.pid")

	os.WriteFile(pidPath, []byte("not-a-number\n"), 0600)

	pm := NewProcessManager()
	_, err := pm.GetPIDFromFile(pidPath)
	if err == nil {
		t.Error("expected error for invalid PID content")
	}
}

func TestDetect3proxy(t *testing.T) {
	pm := NewProcessManager()
	path, err := pm.Detect3proxy()
	_ = path
	_ = err
}

func TestFindBinary(t *testing.T) {
	pm := NewProcessManager()

	path := pm.FindBinary("nonexistent-binary-12345")
	if path != "" {
		t.Errorf("expected empty string for non-existent binary, got %s", path)
	}

	path = pm.FindBinary("ls")
	if path == "" {
		t.Skip("ls not found in PATH")
	}
	if !strings.HasSuffix(path, "ls") {
		t.Errorf("expected path ending with ls, got %s", path)
	}
}

func TestBinaryName(t *testing.T) {
	name := BinaryName()

	switch runtime.GOOS {
	case "windows":
		if name != "3proxy.exe" {
			t.Errorf("expected 3proxy.exe on Windows, got %s", name)
		}
	case "darwin":
		if name != "3proxy-darwin-"+runtime.GOARCH {
			t.Errorf("expected 3proxy-darwin-ARCH on macOS, got %s", name)
		}
	case "linux":
		if name != "3proxy-linux-"+runtime.GOARCH {
			t.Errorf("expected 3proxy-linux-ARCH on Linux, got %s", name)
		}
	}
}

func TestInstallPath(t *testing.T) {
	pm := NewProcessManager()
	path := pm.InstallPath()

	if path == "" {
		t.Skip("UserHomeDir not available")
	}

	if runtime.GOOS == "windows" {
		if !strings.Contains(path, "AppData") {
			t.Errorf("expected Windows path to contain AppData, got %s", path)
		}
	} else {
		if !strings.Contains(path, ".local") {
			t.Errorf("expected path to contain .local, got %s", path)
		}
	}
}

func TestConfigPath(t *testing.T) {
	pm := NewProcessManager()
	path := pm.ConfigPath("test-pool-123")

	if path == "" {
		t.Skip("UserHomeDir not available")
	}

	if !strings.Contains(path, "test-pool-123") {
		t.Errorf("expected path to contain pool ID, got %s", path)
	}
}

func TestPIDPath(t *testing.T) {
	pm := NewProcessManager()
	path := pm.PIDPath("my-pool")

	if path == "" {
		t.Skip("UserHomeDir not available")
	}

	if !strings.Contains(path, "3proxy.pid") {
		t.Errorf("expected path to end with 3proxy.pid, got %s", path)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, "a", "b", "c")

	err := EnsureDir(nestedPath)
	if err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}
}

func TestProcess_Wait(t *testing.T) {
	pm := NewProcessManager()

	proc, err := pm.Start("")
	if err == nil {
		defer pm.Stop(proc.PID)
		defer proc.Wait()

		if proc.PID <= 0 {
			t.Errorf("expected valid PID, got %d", proc.PID)
		}
	} else {
		t.Skipf("Cannot start test process: %v", err)
	}
}

func TestProcess_Signal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Signal test not supported on Windows")
	}

	pm := NewProcessManager()

	proc, err := pm.Start("")
	if err != nil {
		t.Skipf("Cannot start test process: %v", err)
	}
	defer pm.Stop(proc.PID)

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		t.Logf("SIGTERM returned error (expected): %v", err)
	}
}

func TestStop_InvalidPID(t *testing.T) {
	pm := NewProcessManager()

	err := pm.Stop(-1)
	if err == nil {
		t.Error("expected error for invalid PID")
	}

	err = pm.Stop(0)
	if err == nil {
		t.Error("expected error for PID 0")
	}
}

func TestStop_NonExistent(t *testing.T) {
	pm := NewProcessManager()

	err := pm.Stop(999999999)
	if err != nil {
		t.Logf("Stop on non-existent process: %v", err)
	}
}

func TestStart_EmptyPath(t *testing.T) {
	pm := NewProcessManager()

	_, err := pm.Start("")
	if err != nil {
		t.Logf("Start with empty path (3proxy not found): %v", err)
	}
}

func TestStart_CustomPath(t *testing.T) {
	pm := NewProcessManager()

	nonexistentPath := filepath.Join(t.TempDir(), "nonexistent-binary")
	_, err := pm.Start(nonexistentPath, "--invalid-arg")
	if err == nil {
		t.Error("expected error starting non-existent binary")
	}
}
