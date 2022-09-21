// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opamp

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/open-telemetry/opamp-go/protobufs"
)

type config map[string]interface{}

func (c *config) composeOtConfig() (map[string]interface{}, error) {
	k := koanf.New(".")
	provider := confmap.Provider(*c, ".")
	if err := k.Load(provider, nil); err != nil {
		return nil, err
	}

	// Bindplane returning configuration that doesn't work with our OT distribution.
	k.Delete("labels")

	return k.Raw(), nil
}

func (c *config) composeEffectiveConfigProto() (*protobufs.EffectiveConfig, error) {
	bytes, err := yaml.Parser().Marshal(*c)
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

func (c *config) validate() error {
	// TODO: validate config
	return nil
}
