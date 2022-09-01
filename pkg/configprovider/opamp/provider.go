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
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

const (
	schemeName = "opamp"
)

type provider struct{}

func New() confmap.Provider {
	return &provider{}
}

func (fmp *provider) Retrieve(ctx context.Context, uri string, _ confmap.WatcherFunc) (confmap.Retrieved, error) {
	if !strings.HasPrefix(uri, schemeName+":") {
		return confmap.Retrieved{}, fmt.Errorf("%q uri is not supported by %q provider", uri, schemeName)
	}

	opampEndpoint := uri[len(schemeName)+1:]

	stringMap := map[string]interface{}{
		"receivers": map[string]interface{}{
			"hostmetrics": map[string]interface{}{
				"collection_interval": "30s",
				"scrapers": map[string]interface{}{
					"load": nil,
				},
			},
		},
		"exporters": map[string]interface{}{
			"logging": map[string]interface{}{
				"logLevel": "debug",
			},
		},
		"service": map[string]interface{}{
			"pipelines": map[string]interface{}{
				"metrics": map[string]interface{}{
					"receivers": []string{"hostmetrics"},
					"exporters": []string{"logging"},
				},
			},
		},
	}
	conf := confmap.NewFromStringMap(stringMap)

	opamp, err := newOpAMP(opampEndpoint)

	if err != nil {
		return confmap.Retrieved{}, err
	}

	err = opamp.Start(ctx)

	if err != nil {
		return confmap.Retrieved{}, err
	}

	return confmap.NewRetrieved(conf.ToStringMap())
}

func (*provider) Scheme() string {
	return schemeName
}

func (fmp *provider) Shutdown(context.Context) error {
	return nil
}
