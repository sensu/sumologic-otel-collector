package opamp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type agentState struct {
	InstanceId string `json:"instance_id"`
}

type stateManager struct {
	mu        sync.RWMutex
	statePath string
}

// TODO: set default state path to something other than a temp dir
func (m *stateManager) StatePath() string {
	if m.statePath == "" {
		return filepath.Join(os.TempDir(), "ot-opamp-state")
	}
	return m.statePath
}

func (m *stateManager) Load() (agentState, error) {
	var state agentState

	m.mu.Lock()
	data, err := os.ReadFile(m.StatePath())
	if err != nil {
		m.mu.Unlock()
		return state, err
	}
	m.mu.Unlock()

	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	return state, nil
}

func (m *stateManager) Save(state agentState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	f, err := os.OpenFile(m.StatePath(), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	bytes, err := json.Marshal(state)
	if err != nil {
		return err
	}

	_, err = f.Write(bytes)

	return err
}
