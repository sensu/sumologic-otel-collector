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
	"errors"
	"fmt"
	"sync"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type jobsreceiver struct {
	config *Config

	sync.Mutex
	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
	cancel    context.CancelFunc

	logsConsumer      consumer.Logs
	metricsConsumer   consumer.Metrics
	consumeRetryDelay time.Duration
	consumeMaxRetries uint64

	logger *zap.Logger

	jobLogs    chan plog.Logs
	jobMetrics chan pmetric.Metrics
}

func NewJobsReceiver() *jobsreceiver {
	return &jobsreceiver{
		jobLogs:    make(chan plog.Logs),
		jobMetrics: make(chan pmetric.Metrics),
	}
}

// Ensure this receiver adheres to required interface.
var _ component.MetricsReceiver = (*jobsreceiver)(nil)

// Ensure this receiver adheres to required interface.
var _ component.LogsReceiver = (*jobsreceiver)(nil)

// Start tells the receiver to start.
func (r *jobsreceiver) Start(ctx context.Context, host component.Host) error {
	r.logger.Info("Starting jobs receiver")

	r.Lock()
	defer r.Unlock()

	r.startOnce.Do(func() {
		rctx, cancel := context.WithCancel(ctx)
		r.cancel = cancel

		r.wg.Add(1)
		go func() {
			var lErr error
			defer r.wg.Done()
			for {
				select {
				case <-rctx.Done():
					return
				case l := <-r.jobLogs:
					lErr = r.consumeLogsWithRetry(rctx, l)
					if lErr != nil {
						r.logger.Error("ConsumeLogs() error",
							zap.String("error", lErr.Error()),
						)
					}
				}
			}
		}()

		r.wg.Add(1)
		go func() {
			var mErr error
			defer r.wg.Done()
			for {
				select {
				case <-rctx.Done():
					return
				case m := <-r.jobMetrics:
					mErr = r.consumeMetricsWithRetry(rctx, m)
					if mErr != nil {
						r.logger.Error("ConsumeMetrics() error",
							zap.String("error", mErr.Error()),
						)
					}
				}
			}
		}()

		r.scheduleJobs(rctx)
	})

	return nil
}

// Consume logs and retry on recoverable errors
func (r *jobsreceiver) consumeLogsWithRetry(ctx context.Context, logs plog.Logs) error {
	constantBackoff := backoff.WithMaxRetries(backoff.NewConstantBackOff(r.consumeRetryDelay), r.consumeMaxRetries)

	// retry handling according to https://github.com/open-telemetry/opentelemetry-collector/blob/master/component/receiver.go#L45
	err := backoff.RetryNotify(
		func() error {
			// we need to check for context cancellation here
			select {
			case <-ctx.Done():
				return backoff.Permanent(errors.New("closing"))
			default:
			}
			err := r.logsConsumer.ConsumeLogs(ctx, logs)
			if consumererror.IsPermanent(err) {
				return backoff.Permanent(err)
			} else {
				return err
			}
		},
		constantBackoff,
		func(err error, delay time.Duration) {
			r.logger.Warn("ConsumeLogs() recoverable error, will retry",
				zap.Error(err), zap.Duration("delay", delay),
			)
		},
	)

	return err
}

// Consume metrics and retry on recoverable errors
func (r *jobsreceiver) consumeMetricsWithRetry(ctx context.Context, metrics pmetric.Metrics) error {
	constantBackoff := backoff.WithMaxRetries(backoff.NewConstantBackOff(r.consumeRetryDelay), r.consumeMaxRetries)

	// retry handling according to https://github.com/open-telemetry/opentelemetry-collector/blob/master/component/receiver.go#L45
	err := backoff.RetryNotify(
		func() error {
			// we need to check for context cancellation here
			select {
			case <-ctx.Done():
				return backoff.Permanent(errors.New("closing"))
			default:
			}
			err := r.metricsConsumer.ConsumeMetrics(ctx, metrics)
			if consumererror.IsPermanent(err) {
				return backoff.Permanent(err)
			} else {
				return err
			}
		},
		constantBackoff,
		func(err error, delay time.Duration) {
			r.logger.Warn("ConsumeMetrics() recoverable error, will retry",
				zap.Error(err), zap.Duration("delay", delay),
			)
		},
	)

	return err
}

func (r *jobsreceiver) scheduleJobs(context.Context) error {
	for _, j := range r.config.Jobs {
		fmt.Println(j)
		go func() { r.jobLogs <- plog.Logs{} }()
		go func() { r.jobMetrics <- pmetric.Metrics{} }()
	}

	return nil
}

// Shutdown is invoked during service shutdown.
func (r *jobsreceiver) Shutdown(context.Context) error {
	r.Lock()
	defer r.Unlock()

	r.stopOnce.Do(func() {
		r.logger.Info("Stopping jobs receiver")
		r.cancel()
		r.wg.Wait()
	})

	return nil
}
