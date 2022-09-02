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

const localConfig = `
exporters:
  otlp:
    endpoint: localhost:1111

receivers:
  otlp:
    protocols:
      grpc: {}
      http: {}

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: []
      exporters: [otlp]
`

type Agent struct {
	logger types.Logger

	agentType    string
	agentVersion string

	serverURL string

	effectiveConfig string

	instanceId ulid.ULID

	agentDescription *protobufs.AgentDescription

	opampClient client.OpAMPClient

	remoteConfigStatus *protobufs.RemoteConfigStatus
}

func newAgent(logger types.Logger, serverURL string) *Agent {
	agent := &Agent{
		logger:          logger,
		agentType:       "sumologic-otel-collector",
		agentVersion:    "0.0.1",
		serverURL:       serverURL,
		effectiveConfig: localConfig,
	}

	agent.createAgentIdentity()

	agent.setRemoteConfigStatus()

	agent.loadLocalConfig()

	return agent
}

func (agent *Agent) Start() error {
	agent.logger.Debugf("Agent starting, id=%v, type=%s, version=%s.",
		agent.instanceId.String(), agent.agentType, agent.agentVersion)

	agent.opampClient = client.NewWebSocket(agent.logger)

	settings := types.StartSettings{
		OpAMPServerURL: agent.serverURL,
		Header: http.Header{
			"Authorization":  []string{fmt.Sprintf("Secret-Key %s", "foobar")},
			"User-Agent":     []string{fmt.Sprintf("sumologic-otel-collector/%s", "0.0.1")},
			"OpAMP-Version":  []string{"v0.2.0"}, // BindPlane currently requires OpAMP 0.2.0
			"Agent-ID":       []string{"foo"},
			"Agent-Version":  []string{"0.0.1"},
			"Agent-Hostname": []string{"stealth"},
		},
		InstanceUid: agent.instanceId.String(),
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

func stringKeyValue(key, value string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key: key,
		Value: &protobufs.AnyValue{
			Value: &protobufs.AnyValue_StringValue{StringValue: value},
		},
	}
}

func (agent *Agent) createAgentIdentity() {
	// Generate instance id.
	entropy := ulid.Monotonic(rand.New(rand.NewSource(0)), 0)
	agent.instanceId = ulid.MustNew(ulid.Timestamp(time.Now()), entropy)

	hostname, _ := os.Hostname()

	ident := []*protobufs.KeyValue{
		stringKeyValue("service.instance.id", agent.instanceId.String()),
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

func (agent *Agent) setRemoteConfigStatus() {
	agent.remoteConfigStatus = &protobufs.RemoteConfigStatus{
		Status: protobufs.RemoteConfigStatus_UNSET,
	}
}

func (agent *Agent) updateAgentIdentity(instanceId ulid.ULID) {
	agent.logger.Debugf("Agent identify is being changed from id=%v to id=%v",
		agent.instanceId.String(),
		instanceId.String())
	agent.instanceId = instanceId
}

func (agent *Agent) loadLocalConfig() {
	var k = koanf.New(".")
	_ = k.Load(rawbytes.Provider([]byte(localConfig)), yaml.Parser())

	effectiveConfigBytes, err := k.Marshal(yaml.Parser())
	if err != nil {
		panic(err)
	}

	agent.effectiveConfig = string(effectiveConfigBytes)
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

	// Begin with local config. We will later merge received configs on top of it.
	var k = koanf.New(".")
	if err := k.Load(rawbytes.Provider([]byte(localConfig)), yaml.Parser()); err != nil {
		return false, err
	}

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
	fmt.Println("Received message from the OpAMP server.\n")
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
	}
}
