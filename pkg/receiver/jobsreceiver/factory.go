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
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
)

const (
	typeStr        = "jobs"
	versionStr     = "v0.1"
	stabilityLevel = component.StabilityLevelAlpha
)

// NewFactory creates a factory for jobs receiver.
func NewFactory() component.ReceiverFactory {
	jr := &jobsreceiver{}
	return component.NewReceiverFactory(
		typeStr,
		createDefaultConfig,
		component.WithLogsReceiver(createLogsReceiver(jr), stabilityLevel),
		component.WithMetricsReceiver(createMetricsReceiver(jr), stabilityLevel),
	)
}

func createDefaultConfig() config.Receiver {
	rs := config.NewReceiverSettings(config.NewComponentID(typeStr))

	return &Config{
		ReceiverSettings:  &rs,
		ConsumeRetryDelay: 500 * time.Millisecond,
		ConsumeMaxRetries: 10,
		Job: jobConfig{
			Name: "default",
			Schedule: jobScheduleConfig{
				Interval: 10,
			},
			Exec: jobExecConfig{
				Command: "echo -n foobar",
			},
		},
	}
}

func createLogsReceiver(jr *jobsreceiver) component.CreateLogsReceiverFunc {
	return func(
		ctx context.Context,
		params component.ReceiverCreateSettings,
		cfg config.Receiver,
		nextConsumer consumer.Logs) (component.LogsReceiver, error) {

		tCfg, _ := cfg.(*Config)

		jr.logsConsumer = nextConsumer
		jr.logger = params.Logger
		jr.consumeRetryDelay = tCfg.ConsumeRetryDelay
		jr.consumeMaxRetries = tCfg.ConsumeMaxRetries

		return jr, nil
	}
}

func createMetricsReceiver(jr *jobsreceiver) component.CreateMetricsReceiverFunc {
	return func(
		ctx context.Context,
		params component.ReceiverCreateSettings,
		cfg config.Receiver,
		nextConsumer consumer.Metrics) (component.MetricsReceiver, error) {

		tCfg, _ := cfg.(*Config)

		jr.metricsConsumer = nextConsumer
		jr.logger = params.Logger
		jr.consumeRetryDelay = tCfg.ConsumeRetryDelay
		jr.consumeMaxRetries = tCfg.ConsumeMaxRetries

		return jr, nil
	}
}
