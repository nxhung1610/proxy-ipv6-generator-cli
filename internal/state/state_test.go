package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

func TestStateManager_CreateAndGet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm, err := New(path)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	state := sm.GetState()
	if state.Version != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %q", state.Version)
	}
	if state.Pools == nil {
		t.Error("expected non-nil Pools slice")
	}
}

func TestStateManager_Create_InitializesEmptyState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm, err := New(path)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Persist the initial state to disk
	if err := sm.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// File should be created
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	var state types.State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("invalid JSON in state file: %v", err)
	}

	if state.Version != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %q", state.Version)
	}
}

func TestStateManager_UpdatePool_Add(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm, _ := New(path)

	pool := types.Pool{
		ID:     "pool-1",
		Name:   "test-pool",
		Prefix: "2001:db8::/64",
		Status: string(types.PoolStatusRunning),
	}

	if err := sm.UpdatePool(pool); err != nil {
		t.Fatalf("UpdatePool failed: %v", err)
	}

	state := sm.GetState()
	if len(state.Pools) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(state.Pools))
	}
	if state.Pools[0].Name != "test-pool" {
		t.Errorf("expected pool name %q, got %q", "test-pool", state.Pools[0].Name)
	}
}

func TestStateManager_UpdatePool_Replace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm, _ := New(path)

	initial := types.Pool{ID: "pool-1", Name: "test-pool", Status: string(types.PoolStatusStopped)}
	if err := sm.UpdatePool(initial); err != nil {
		t.Fatalf("UpdatePool failed: %v", err)
	}

	updated := types.Pool{ID: "pool-1", Name: "test-pool", Status: string(types.PoolStatusRunning)}
	if err := sm.UpdatePool(updated); err != nil {
		t.Fatalf("UpdatePool update failed: %v", err)
	}

	state := sm.GetState()
	if len(state.Pools) != 1 {
		t.Fatalf("expected 1 pool after update, got %d", len(state.Pools))
	}
	if state.Pools[0].Status != string(types.PoolStatusRunning) {
		t.Errorf("expected status %q, got %q", string(types.PoolStatusRunning), state.Pools[0].Status)
	}
}

func TestStateManager_UpdatePool_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm1, _ := New(path)
	if err := sm1.UpdatePool(types.Pool{ID: "pool-1", Name: "persist-pool"}); err != nil {
		t.Fatalf("UpdatePool failed: %v", err)
	}

	// Simulate a new process: new StateManager reads from same path
	sm2, err := New(path)
	if err != nil {
		t.Fatalf("New (second process) failed: %v", err)
	}

	state := sm2.GetState()
	if len(state.Pools) != 1 {
		t.Errorf("expected 1 persisted pool, got %d", len(state.Pools))
	}
	if state.Pools[0].Name != "persist-pool" {
		t.Errorf("expected persisted pool name %q, got %q", "persist-pool", state.Pools[0].Name)
	}
}

func TestStateManager_RemovePool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm, _ := New(path)
	if err := sm.UpdatePool(types.Pool{ID: "pool-1"}); err != nil {
		t.Fatalf("UpdatePool failed: %v", err)
	}
	if err := sm.UpdatePool(types.Pool{ID: "pool-2"}); err != nil {
		t.Fatalf("UpdatePool failed: %v", err)
	}

	if err := sm.RemovePool("pool-1"); err != nil {
		t.Fatalf("RemovePool failed: %v", err)
	}

	state := sm.GetState()
	if len(state.Pools) != 1 {
		t.Errorf("expected 1 pool after removal, got %d", len(state.Pools))
	}
	if state.Pools[0].ID != "pool-2" {
		t.Errorf("expected remaining pool to be pool-2, got %q", state.Pools[0].ID)
	}
}

func TestStateManager_RemovePool_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm1, _ := New(path)
	if err := sm1.UpdatePool(types.Pool{ID: "pool-1", Name: "to-delete"}); err != nil {
		t.Fatalf("UpdatePool failed: %v", err)
	}
	if err := sm1.UpdatePool(types.Pool{ID: "pool-2", Name: "to-keep"}); err != nil {
		t.Fatalf("UpdatePool failed: %v", err)
	}
	if err := sm1.RemovePool("pool-1"); err != nil {
		t.Fatalf("RemovePool failed: %v", err)
	}

	// New process reads persisted state
	sm2, _ := New(path)
	state := sm2.GetState()
	if len(state.Pools) != 1 {
		t.Errorf("expected 1 pool after remove+restart, got %d", len(state.Pools))
	}
}

func TestStateManager_RemovePool_NoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm, _ := New(path)
	if err := sm.RemovePool("non-existent"); err != nil {
		t.Errorf("RemovePool for non-existent should not error, got: %v", err)
	}
}

func TestStateManager_SetProxyState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm, _ := New(path)

	proxyState := types.ThreeProxyState{
		PID:        12345,
		ConfigPath: "/tmp/3proxy.cfg",
		StartedAt:  "2026-05-09T19:50:00Z",
	}

	if err := sm.SetProxyState(proxyState); err != nil {
		t.Fatalf("SetProxyState failed: %v", err)
	}

	state := sm.GetState()
	if state.Proxy3.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", state.Proxy3.PID)
	}
}

func TestStateManager_SetProxyState_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	sm1, _ := New(path)
	if err := sm1.SetProxyState(types.ThreeProxyState{PID: 99999, ConfigPath: "/test.cfg"}); err != nil {
		t.Fatalf("SetProxyState failed: %v", err)
	}

	sm2, _ := New(path)
	state := sm2.GetState()
	if state.Proxy3.PID != 99999 {
		t.Errorf("expected persisted PID 99999, got %d", state.Proxy3.PID)
	}
}

func TestStateManager_PathDirectoryCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "path", "state.json")

	_, err := New(path)
	if err != nil {
		t.Fatalf("New failed for nested path: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		t.Error("expected parent directory to be created")
	}
}

func TestStateManager_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent test in short mode")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	sm, err := New(path)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = sm.UpdatePool(types.Pool{ID: fmt.Sprintf("pool-%d", id)}) //nolint:errcheck
		}(i)
	}
	wg.Wait()

	state := sm.GetState()
	if len(state.Pools) != 50 {
		t.Errorf("expected 50 pools, got %d", len(state.Pools))
	}
}

func TestStateManager_CorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	os.WriteFile(path, []byte("{ invalid json }"), 0644) //nolint:errcheck

	sm, err := New(path)
	if err != nil {
		t.Fatalf("should not error on corrupted file: %v", err)
	}

	state := sm.GetState()
	if state.Version != "0.1.0" {
		t.Errorf("expected reinitialized version, got: %s", state.Version)
	}
	if len(state.Pools) != 0 {
		t.Errorf("expected empty pools on corrupted init, got: %d", len(state.Pools))
	}
}

func TestStateManager_ConcurrentReads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	sm, _ := New(path)
	_ = sm.UpdatePool(types.Pool{ID: "test-pool"}) //nolint:errcheck

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.GetState()
		}()
	}
	wg.Wait()
}

func TestStateManager_ConcurrentWriteAndRead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	sm, _ := New(path)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = sm.UpdatePool(types.Pool{ID: fmt.Sprintf("pool-%d", id)}) //nolint:errcheck
		}(i)
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = sm.GetState()
		}(i)
	}
	wg.Wait()
}
