package opamp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/oklog/ulid/v2"
	"github.com/open-telemetry/opamp-go/protobufs"
)

type EmptyInstanceIdError struct{}

func (e *EmptyInstanceIdError) Error() string {
	return "instance id is empty"
}

type agentState struct {
	InstanceId      string                 `json:"instance_id"`
	EffectiveConfig map[string]interface{} `json:"effective_config"`
}

func newAgentState() *agentState {
	fmt.Println("newAgentState")

	k := koanf.New(".")

	fmt.Printf("koanf: %+v\n", k.Raw())

	return &agentState{
		InstanceId:      newInstanceId(),
		EffectiveConfig: k.Raw(),
	}
}

func (s *agentState) composeEffectiveConfig() (*protobufs.EffectiveConfig, error) {
	fmt.Println("composeEffectiveConfig")

	bytes, err := yaml.Parser().Marshal(s.EffectiveConfig)
	if err != nil {
		return nil, err
	}

	return &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"": {Body: bytes},
			},
		},
	}, nil
}

func (s *agentState) validate() error {
	fmt.Println("validate")
	if s.InstanceId == "" {
		return &EmptyInstanceIdError{}
	}
	return nil
}

func newInstanceId() string {
	fmt.Println("newInstanceId")
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
	fmt.Println("State Path:", m.StatePath())

	m.mu.Lock()
	fmt.Println("LOAD 1")
	data, err := os.ReadFile(m.StatePath())
	if err != nil {
		fmt.Println("LOAD 1 ERR:", err.Error())
		m.mu.Unlock()
		return nil, err
	}
	m.mu.Unlock()
	fmt.Println("LOAD 2")

	var state *agentState
	if err := json.Unmarshal(data, &state); err != nil {
		fmt.Println("LOAD 2 ERR:", err.Error())
		return nil, err
	}

	fmt.Println("LOAD 3")
	if err := state.validate(); err != nil {
		fmt.Println("LOAD 3 ERR:", err.Error())
		return nil, err
	}

	fmt.Println("LOAD 4")
	return state, nil
}

func (m *stateManager) Save(state *agentState) error {
	fmt.Println("Save 1")
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

	fmt.Println("Wrote state to:", m.StatePath())

	return err
}
