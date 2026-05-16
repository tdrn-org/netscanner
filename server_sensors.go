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
	"fmt"
	"log/slog"

	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/sensor"
)

func (s *Server) AddSensor(ctx context.Context, config *SensorConfig) (*sensor.Sensor, error) {
	if config.SyslogSensor != nil {
		return s.addSyslogSensor(ctx, config.SyslogSensor)
	}
	if config.LogfileSensor != nil {
		return s.addLogfileSensor(ctx, config.LogfileSensor)
	}
	if config.AccesslogSensor != nil {
		return s.addAccesslogSensor(ctx, config.AccesslogSensor)
	}
	return nil, fmt.Errorf("empty sensor configuration")
}

func (s *Server) resolveLogMatcherIndex(ctx context.Context, name string) (*logmatcher.Index, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.resolveLogMatcherLocked(ctx, name)
}

func (s *Server) resolveLogMatcherLocked(ctx context.Context, name string) (*logmatcher.Index, error) {
	index := s.logMatchers[name]
	if index != nil {
		return index, nil
	}
	indexModel, err := s.store.SelectOrInsertLogMatcherIndex(ctx, name)
	if err != nil {
		return nil, err
	}
	index = indexModel.ToIndex()
	s.logMatchers[name] = index
	return index, nil
}

func (s *Server) eventReceiver() sensor.EventReceiver {
	return sensor.EventReceiverFunc(s.queueEvent)
}

func (s *Server) queueEvent(ctx context.Context, event *sensor.Event) {
	if event.Host == "" {
		event.Host = s.defaultHost
	}
	if event.Service == "" {
		event.Service = event.Host
	}
	s.recordEventInfos(ctx, event)
	if event.Type != sensor.EventTypeInformational {
		s.recordEvent(ctx, event)
	}
}

func (s *Server) recordEventInfos(ctx context.Context, event *sensor.Event) {
	if event.HardwareAddress != nil {
		s.arpCache.Put(ctx, event.Address, event.HardwareAddress)
	}
}

func (s *Server) recordEvent(ctx context.Context, event *sensor.Event) {
	s.logger.Info(event.String())
	deviceInfo := s.deviceInfos.Lookup(ctx, event.Address)
	s.logger.Info(deviceInfo.String())
	err := s.store.UpdateOrInsertEvent(ctx, event, deviceInfo)
	if err != nil {
		s.logger.Error("failed to record event", slog.Any("err", err))
	}
	s.metricsRecorder.RecordEvent(event, deviceInfo)
}
