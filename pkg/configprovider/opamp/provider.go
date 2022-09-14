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

	"github.com/knadh/koanf"
	kconfmap "github.com/knadh/koanf/providers/confmap"
	"go.opentelemetry.io/collector/confmap"
)

const (
	schemeName = "opamp"
)

type provider struct {
	opampAgent *Agent
}

func New() confmap.Provider {
	fmt.Println("New")

	logger := &Logger{log.Default()}

	return &provider{
		opampAgent: newAgent(logger),
	}
}

func (fmp *provider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (confmap.Retrieved, error) {
	fmt.Println("Retrieve")

	if !strings.HasPrefix(uri, schemeName+":") {
		return confmap.Retrieved{}, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}
	opampEndpoint := uri[len(schemeName)+1:]

	if err := fmp.opampAgent.Start(opampEndpoint); err != nil {
		return confmap.Retrieved{}, err
	}

	fmt.Println("BEFORE INITIAL CONFIG UPDATED")
	<-fmp.opampAgent.configUpdated
	fmt.Println("AFTER INITIAL CONFIG UPDATED")

	go func() {
		<-fmp.opampAgent.configUpdated
		fmt.Println("CONFIG UPDATED!")
		watcher(&confmap.ChangeEvent{})
	}()

	k := koanf.New(".")
	if err := k.Load(kconfmap.Provider(fmp.opampAgent.state.EffectiveConfig, "."), nil); err == nil {
		return confmap.Retrieved{}, err
	}
	k.Delete("labels")

	return confmap.NewRetrieved(k.Raw())
}

func (*provider) Scheme() string {
	fmt.Println("Scheme")
	return schemeName
}

func (fmp *provider) Shutdown(context.Context) error {
	fmt.Println("Shutdown")
	fmp.opampAgent.Shutdown()

	return nil
}
