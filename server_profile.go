/*
 * Copyright 2025-2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package netscanner

import (
	"context"
	"log/slog"
	"sync"

	"github.com/tdrn-org/netscanner/internal/datastore/model"
	"github.com/tdrn-org/netscanner/sensor"
)

type ServerProfile interface {
	Name() string
	Start(ctx context.Context)
	Shutdown(ctx context.Context) error
	Close() error
}

type serverProfile struct {
	receiver sensor.EventReceiver
	model    *model.Profile
	started  bool
	mutex    sync.Mutex
	logger   *slog.Logger
}

func (p *serverProfile) Name() string {
	return p.model.Name
}

func (p *serverProfile) Start(ctx context.Context) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.started {
		p.logger.Debug("already started")
		return
	}
	p.logger.Info("starting...")
	for _, syslogSensor := range p.model.SyslogSensors {
		p.startSyslogSensor(syslogSensor)
	}
	p.started = true
	p.logger.Info("started")
}

func (p *serverProfile) startSyslogSensor(syslogSensor *model.SyslogSensor) {
	if !syslogSensor.Enabled {
		p.logger.Debug("skipping disabled syslog sensor", slog.String("sensor", syslogSensor.Name))
		return
	}
}

func (p *serverProfile) Shutdown(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.started {
		p.logger.Debug("already stopped")
		return nil
	}
	return nil
}

func (p *serverProfile) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return nil
}
