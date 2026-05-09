package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/nxhung/proxy-ipv6-generator-cli/pkg/types"
)

type StateManager struct {
	mu    sync.RWMutex
	state *types.State
	path  string
}

func New(path string) (*StateManager, error) {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		switch {
		case os.Getenv("OS") == "Windows_NT":
			path = filepath.Join(os.Getenv("LOCALAPPDATA"), "proxy-ipv6-cli", "state.json")
		default:
			path = filepath.Join(homeDir, ".local", "share", "proxy-ipv6-cli", "state.json")
		}
	}

	// Sanitize path: resolve . and .. components to prevent traversal
	cleanPath := filepath.Clean(path)

	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	sm := &StateManager{path: cleanPath}
	sm.state = &types.State{
		Version: "0.1.0",
		Pools:   []types.Pool{},
	}

	if err := sm.load(); err != nil {
		// Reinitialize on load error — prevents nil pointer if file is corrupted
		sm.state = &types.State{
			Version: "0.1.0",
			Pools:   []types.Pool{},
		}
	}

	return sm, nil
}

func (sm *StateManager) load() error {
	data, err := os.ReadFile(sm.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var state types.State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	sm.state = &state
	return nil
}

func (sm *StateManager) Save() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(sm.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (sm *StateManager) GetState() *types.State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

func (sm *StateManager) UpdatePool(pool types.Pool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, p := range sm.state.Pools {
		if p.ID == pool.ID {
			sm.state.Pools[i] = pool
			return sm.saveLocked()
		}
	}

	sm.state.Pools = append(sm.state.Pools, pool)
	return sm.saveLocked()
}

func (sm *StateManager) RemovePool(poolID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, p := range sm.state.Pools {
		if p.ID == poolID {
			sm.state.Pools = append(sm.state.Pools[:i], sm.state.Pools[i+1:]...)
			return sm.saveLocked()
		}
	}

	return nil
}

func (sm *StateManager) SetProxyState(state types.ThreeProxyState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.Proxy3 = state
	return sm.saveLocked()
}

func (sm *StateManager) saveLocked() error {
	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.path, data, 0600)
}
