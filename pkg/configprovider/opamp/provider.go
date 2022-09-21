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
	"context"
	"fmt"
	"log"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

const (
	schemeName = "opamp"
)

type provider struct {
	opampAgent *Agent
}

func New() confmap.Provider {
	logger := &Logger{log.Default()}

	return &provider{
		opampAgent: newAgent(logger),
	}
}

func (fmp *provider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return confmap.Retrieved{}, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}
	opampEndpoint := uri[len(schemeName)+1:]

	if err := fmp.opampAgent.loadState(); err != nil {
		return confmap.Retrieved{}, err
	}

	configIsEmpty := true
	if len(fmp.opampAgent.state.EffectiveConfig) > 0 {
		configIsEmpty = false
	}

	if err := fmp.opampAgent.Start(opampEndpoint); err != nil {
		return confmap.Retrieved{}, err
	}

	// If no config was previously loaded loaded, wait for the first config to be
	// received before kicking off the goroutine to listen for config updates.
	if configIsEmpty {
		<-fmp.opampAgent.configUpdated
	}

	// Listen for config updates and trigger a change event to internally reload
	// the ot collector when one is received.
	go func() {
		<-fmp.opampAgent.configUpdated
		watcher(&confmap.ChangeEvent{})
	}()

	config, err := fmp.opampAgent.state.EffectiveConfig.composeOtConfig()
	if err != nil {
		return confmap.Retrieved{}, err
	}

	return confmap.NewRetrieved(config)
}

func (*provider) Scheme() string {
	return schemeName
}

func (fmp *provider) Shutdown(context.Context) error {
	fmp.opampAgent.Shutdown()

	return nil
}
