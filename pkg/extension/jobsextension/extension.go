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

package jobsextension

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.uber.org/zap"
)

type JobsExtension struct {
	logger *zap.Logger
	conf   *Config
}

func newJobsExtension(conf *Config, logger *zap.Logger) (*JobsExtension, error) {
	return &JobsExtension{
		conf:   conf,
		logger: logger,
	}, nil
}

func (je *JobsExtension) runJob(job jobConfig) error {
	je.logger.Info("Running Monitoring Job!")

	cmd := exec.Command("sh", "-c", job.Exec.Command)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	output, err := io.ReadAll(stdout)

	if err != nil {
		return err
	}

	fmt.Println(string(output))

	return nil
}

func (je *JobsExtension) Start(ctx context.Context, host component.Host) error {
	err := je.runJob(je.conf.Job)
	return err
}

func (je *JobsExtension) Shutdown(ctx context.Context) error {
	return nil
}

func (je *JobsExtension) ComponentID() config.ComponentID {
	return je.conf.ExtensionSettings.ID()
}
