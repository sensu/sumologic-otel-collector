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

package opamp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/client/types"
	"go.uber.org/zap"
)

const (
	yamlContentType = "text/yaml"
)

// TEMPORARY: This only exists to allow the creation of the OpAmp client for now.
type opAMPLogger struct {
	logger *zap.Logger
}

func newOpAMPLogger(logger *zap.Logger) *opAMPLogger {
	return &opAMPLogger{
		logger: logger,
	}
}

func (l *opAMPLogger) Debugf(format string, v ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}

func (l *opAMPLogger) Errorf(format string, v ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

type OpAMP struct {
	logger     *zap.Logger
	instanceid string
	endpoint   string
	client     client.OpAMPClient
	cancel     context.CancelFunc
}

func newOpAMP(endpoint string) (*OpAMP, error) {
	logger, _ := zap.NewProduction()

	// Unfortunately, we don't seem to have access to original instance uuid so need to use another one.
	instanceuuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return &OpAMP{
		logger:     logger,
		endpoint:   endpoint,
		instanceid: instanceuuid.String(),
	}, nil
}

func (se *OpAMP) Start(ctx context.Context) error {
	se.logger.Info("Starting OpAMP Client")
	settings := types.StartSettings{
		OpAMPServerURL: se.endpoint,
		InstanceUid:    se.instanceid,
		Callbacks: types.CallbacksStruct{
			OnConnectFunc: func() {
				se.logger.Debug("Connected to OpAMP Server")
			},
			OnConnectFailedFunc: func(err error) {
				se.logger.Error("Failed to connect to OpAMP Server", zap.Error(err))
			},
			OnMessageFunc: func(ctx context.Context, msg *types.MessageData) {
				se.logger.Debug("Received OpAMP message")
			},
		},
	}

	se.client = client.NewWebSocket(newOpAMPLogger(se.logger))
	var err error

	err = se.client.Start(ctx, settings)

	return err
}

func (se *OpAMP) Shutdown(ctx context.Context) error {
	se.cancel()
	return se.client.Stop(ctx)
}
