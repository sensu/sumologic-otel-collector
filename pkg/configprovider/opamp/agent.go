// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package opamp

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/oklog/ulid/v2"

	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
)

type Agent struct {
	logger types.Logger

	agentType    string
	agentVersion string

	serverURL string

	effectiveConfig string

	configUpdated chan bool

	instanceId string

	agentDescription *protobufs.AgentDescription

	opampClient client.OpAMPClient

	remoteConfigStatus *protobufs.RemoteConfigStatus
}

func newAgent(logger types.Logger, serverURL string) *Agent {
	agent := &Agent{
		logger:        logger,
		agentType:     "sumologic-otel-collector",
		agentVersion:  "0.0.1",
		serverURL:     serverURL,
		configUpdated: make(chan bool),
	}

	agent.createAgentId()
	agent.createAgentDescription()

	return agent
}

func (agent *Agent) Start() error {
	agent.logger.Debugf("Agent starting, id=%v, type=%s, version=%s.",
		agent.instanceId, agent.agentType, agent.agentVersion)

	agent.opampClient = client.NewWebSocket(agent.logger)

	hostname, _ := os.Hostname()

	settings := types.StartSettings{
		OpAMPServerURL: agent.serverURL,
		Header: http.Header{
			"Authorization":  []string{fmt.Sprintf("Secret-Key %s", "foobar")},
			"User-Agent":     []string{fmt.Sprintf("sumologic-otel-collector/%s", "0.0.1")},
			"OpAMP-Version":  []string{"v0.2.0"}, // BindPlane currently requires OpAMP 0.2.0
			"Agent-ID":       []string{agent.instanceId},
			"Agent-Version":  []string{"0.0.1"},
			"Agent-Hostname": []string{hostname},
		},
		InstanceUid: agent.instanceId,
		Callbacks: types.CallbacksStruct{
			OnConnectFunc: func() {
				agent.logger.Debugf("Connected to the OpAMP server.")
			},
			OnConnectFailedFunc: func(err error) {
				agent.logger.Errorf("Failed to connect to the server: %v", err)
			},
			OnErrorFunc: func(err *protobufs.ServerErrorResponse) {
				agent.logger.Errorf("Server returned an error response: %v", err.ErrorMessage)
			},
			SaveRemoteConfigStatusFunc: func(_ context.Context, status *protobufs.RemoteConfigStatus) {
				agent.remoteConfigStatus = status
			},
			GetEffectiveConfigFunc: func(ctx context.Context) (*protobufs.EffectiveConfig, error) {
				return agent.composeEffectiveConfig(), nil
			},
			OnMessageFunc: agent.onMessage,
		},
		RemoteConfigStatus: agent.remoteConfigStatus,
		Capabilities: protobufs.AgentCapabilities_AcceptsRemoteConfig |
			protobufs.AgentCapabilities_ReportsRemoteConfig |
			protobufs.AgentCapabilities_ReportsEffectiveConfig,
	}
	err := agent.opampClient.SetAgentDescription(agent.agentDescription)
	if err != nil {
		return err
	}

	agent.logger.Debugf("Starting OpAMP client...")

	err = agent.opampClient.Start(context.Background(), settings)
	if err != nil {
		return err
	}

	agent.logger.Debugf("OpAMP Client started.")

	return nil
}

func instanceIdFilePath() string {
	return filepath.Join(os.TempDir(), "ot-opamp-id")
}

func loadInstanceId() string {
	data, err := os.ReadFile(instanceIdFilePath())
	if err == nil && len(data) > 0 {
		return string(data)
	}
	return ""
}

func (agent *Agent) saveInstanceId() error {
	f, err := os.OpenFile(instanceIdFilePath(), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(agent.instanceId))

	return err
}

func newInstanceId() string {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(0)), 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

func (agent *Agent) createAgentId() {
	agent.instanceId = loadInstanceId()

	if agent.instanceId == "" {
		agent.instanceId = newInstanceId()
		agent.saveInstanceId()
	}
}

func stringKeyValue(key, value string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key: key,
		Value: &protobufs.AnyValue{
			Value: &protobufs.AnyValue_StringValue{StringValue: value},
		},
	}
}

func (agent *Agent) createAgentDescription() {
	hostname, _ := os.Hostname()

	ident := []*protobufs.KeyValue{
		stringKeyValue("service.instance.id", agent.instanceId),
		stringKeyValue("service.instance.name", hostname),
		stringKeyValue("service.name", agent.agentType),
		stringKeyValue("service.version", agent.agentVersion),
	}

	nonIdent := []*protobufs.KeyValue{
		stringKeyValue("os.arch", runtime.GOARCH),
		stringKeyValue("os.details", runtime.GOOS),
		stringKeyValue("os.family", runtime.GOOS),
		stringKeyValue("host.name", hostname),
	}

	agent.agentDescription = &protobufs.AgentDescription{
		IdentifyingAttributes:    ident,
		NonIdentifyingAttributes: nonIdent,
	}
}

func (agent *Agent) updateAgentIdentity(instanceId ulid.ULID) {
	agent.logger.Debugf("Agent identify is being changed from id=%v to id=%v",
		agent.instanceId,
		instanceId.String())
	agent.instanceId = instanceId.String()
	agent.saveInstanceId()
}

func (agent *Agent) composeEffectiveConfig() *protobufs.EffectiveConfig {
	return &protobufs.EffectiveConfig{
		ConfigMap: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"": {Body: []byte(agent.effectiveConfig)},
			},
		},
	}
}

func (agent *Agent) effectiveConfigMap() (map[string]interface{}, error) {
	var k = koanf.New(".")
	err := k.Load(rawbytes.Provider([]byte(agent.effectiveConfig)), yaml.Parser())

	if err != nil {
		return k.Raw(), err
	}

	// Bindplane returning configuration that doesn't work with our OT distribution.
	k.Delete("labels")

	return k.Raw(), nil
}

type agentConfigFileItem struct {
	name string
	file *protobufs.AgentConfigFile
}

type agentConfigFileSlice []agentConfigFileItem

func (a agentConfigFileSlice) Less(i, j int) bool {
	return a[i].name < a[j].name
}

func (a agentConfigFileSlice) Swap(i, j int) {
	t := a[i]
	a[i] = a[j]
	a[j] = t
}

func (a agentConfigFileSlice) Len() int {
	return len(a)
}

func (agent *Agent) applyRemoteConfig(config *protobufs.AgentRemoteConfig) (configChanged bool, err error) {
	if config == nil {
		return false, nil
	}

	agent.logger.Debugf("Received remote config from server, hash=%x.", config.ConfigHash)

	var k = koanf.New(".")
	orderedConfigs := agentConfigFileSlice{}
	for name, file := range config.Config.ConfigMap {
		if name == "" {
			// skip instance config
			continue
		}
		orderedConfigs = append(orderedConfigs, agentConfigFileItem{
			name: name,
			file: file,
		})
	}

	// Sort to make sure the order of merging is stable.
	sort.Sort(orderedConfigs)

	// Append instance config as the last item.
	instanceConfig := config.Config.ConfigMap[""]
	if instanceConfig != nil {
		orderedConfigs = append(orderedConfigs, agentConfigFileItem{
			name: "",
			file: instanceConfig,
		})
	}

	// Merge received configs.
	for _, item := range orderedConfigs {
		var k2 = koanf.New(".")
		err := k2.Load(rawbytes.Provider(item.file.Body), yaml.Parser())
		if err != nil {
			return false, fmt.Errorf("cannot parse config named %s: %v", item.name, err)
		}
		err = k.Merge(k2)
		if err != nil {
			return false, fmt.Errorf("cannot merge config named %s: %v", item.name, err)
		}
	}

	// The merged final result is our effective config.
	effectiveConfigBytes, err := k.Marshal(yaml.Parser())
	if err != nil {
		panic(err)
	}

	newEffectiveConfig := string(effectiveConfigBytes)
	configChanged = false
	if agent.effectiveConfig != newEffectiveConfig {
		agent.logger.Debugf("Effective config changed. Need to report to server.")
		agent.effectiveConfig = newEffectiveConfig
		configChanged = true
	}

	return configChanged, nil
}

func (agent *Agent) Shutdown() {
	agent.logger.Debugf("Agent shutting down...")
	if agent.opampClient != nil {
		_ = agent.opampClient.Stop(context.Background())
	}
}

func (agent *Agent) onMessage(ctx context.Context, msg *types.MessageData) {
	configChanged := false
	if msg.RemoteConfig != nil {
		var err error
		configChanged, err = agent.applyRemoteConfig(msg.RemoteConfig)
		if err != nil {
			agent.opampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
				LastRemoteConfigHash: msg.RemoteConfig.ConfigHash,
				Status:               protobufs.RemoteConfigStatus_FAILED,
				ErrorMessage:         err.Error(),
			})
		} else {
			agent.opampClient.SetRemoteConfigStatus(&protobufs.RemoteConfigStatus{
				LastRemoteConfigHash: msg.RemoteConfig.ConfigHash,
				Status:               protobufs.RemoteConfigStatus_APPLIED,
			})
		}
	}

	if msg.AgentIdentification != nil {
		newInstanceId, err := ulid.Parse(msg.AgentIdentification.NewInstanceUid)
		if err != nil {
			agent.logger.Errorf(err.Error())
		}
		agent.updateAgentIdentity(newInstanceId)
	}

	if configChanged {
		err := agent.opampClient.UpdateEffectiveConfig(ctx)
		if err != nil {
			agent.logger.Errorf(err.Error())
		}

		agent.configUpdated <- true
	}
}
