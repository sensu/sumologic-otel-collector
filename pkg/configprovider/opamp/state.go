package opamp

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

type EmptyInstanceIdError struct{}

func (e *EmptyInstanceIdError) Error() string {
	return "instance id is empty"
}

type agentState struct {
	InstanceId string `json:"instance_id"`
}

func newAgentState() *agentState {
	return &agentState{
		InstanceId: newInstanceId(),
	}
}

func (s *agentState) validate() error {
	if s.InstanceId == "" {
		return &EmptyInstanceIdError{}
	}
	return nil
}

func newInstanceId() string {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(0)), 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
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

func (m *stateManager) Load() (*agentState, error) {
	m.mu.Lock()
	data, err := os.ReadFile(m.StatePath())
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	m.mu.Unlock()

	var state *agentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if err := state.validate(); err != nil {
		return nil, err
	}
	return state, nil
}

func (m *stateManager) Save(state *agentState) error {
	if err := state.validate(); err != nil {
		return err
	}

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
