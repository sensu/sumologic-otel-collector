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

package opampextension

import (
	"go.opentelemetry.io/collector/config"
)

// Config has the configuration for the OpAMP extension.
type Config struct {
	config.ExtensionSettings `mapstructure:"-"`

	// ConfigPath identifies where the config file with the remote configuration needs to be saved
	// It will be used after receiving the updated config from the server
	ConfigPath string `mapstructure:"config_path"`

	// EnsureOpAMP (default=true) makes sure that OpAMP configuration is always present in the configuration
	EnsureOpAMP *bool `mapstructure:"ensure_opamp"`

	// ServerEndpoint identifies the URL of the OpAMP Server
	ServerEndpoint string `mapstructure:"endpoint"`
}
