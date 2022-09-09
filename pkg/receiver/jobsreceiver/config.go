// Copyright 2022, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jobsreceiver

import (
	"time"

	"go.opentelemetry.io/collector/config"
)

// Config has the configuration for Sensu Monitoring Jobs.
type Config struct {
	*config.ReceiverSettings `mapstructure:"-"`

	// ConsumeRetryDelay is the retry delay for recoverable pipeline errors
	// one frequent source of these kinds of errors is the memory_limiter processor
	ConsumeRetryDelay time.Duration `mapstructure:"consume_retry_delay"`

	// ConsumeMaxRetries is the maximum number of retries for recoverable pipeline errors
	ConsumeMaxRetries uint64 `mapstructure:"consume_max_retries"`

	// Start with a single job
	Job jobConfig `mapstructure:"job"`
}

type jobConfig struct {
	Name     string            `mapstructure:"name"`
	Schedule jobScheduleConfig `mapstructure:"schedule"`
	Exec     jobExecConfig     `mapstructure:"type"`
}

type jobScheduleConfig struct {
	Interval int `mapstructure:"interval"`
}

type jobExecConfig struct {
	Command string `mapstructure:"command"`
}
