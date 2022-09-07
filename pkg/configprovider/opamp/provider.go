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
	opampAgent Agent
}

func New() confmap.Provider {
	return &provider{}
}

func (fmp *provider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return confmap.Retrieved{}, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	opampEndpoint := uri[len(schemeName)+1:]

	fmp.opampAgent = *newAgent(&Logger{log.Default()}, opampEndpoint)

	err := fmp.opampAgent.Start()

	if err != nil {
		return confmap.Retrieved{}, err
	}

	<-fmp.opampAgent.configUpdated

	conf, err := fmp.opampAgent.effectiveConfigMap()

	if err != nil {
		return confmap.Retrieved{}, err
	}

	go func() {
		<-fmp.opampAgent.configUpdated
		watcher(&confmap.ChangeEvent{})
	}()

	return confmap.NewRetrieved(conf)
}

func (*provider) Scheme() string {
	return schemeName
}

func (fmp *provider) Shutdown(context.Context) error {
	fmp.opampAgent.Shutdown()

	return nil
}
