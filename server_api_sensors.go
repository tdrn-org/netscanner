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
	"slices"
	"strings"

	"github.com/tdrn-org/netscanner/sensor"
)

func (s *Server) ListSensors() []*SensorInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sensorInfos := make([]*SensorInfo, 0, len(s.sensors))
	for _, sensor := range s.sensors {
		sensorInfo := &SensorInfo{
			Type:         sensor.Type(),
			Name:         sensor.Name(),
			EventCounter: sensor.EventCounter(),
		}
		sensorInfos = append(sensorInfos, sensorInfo)
	}
	slices.SortFunc(sensorInfos, func(si1 *SensorInfo, si2 *SensorInfo) int { return strings.Compare(si1.Name, si2.Name) })
	return sensorInfos
}

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
	if !event.IsValid() {
		s.logger.Warn("ignoring partial event", slog.String("event", event.String()))
		return
	}
	s.recordEventInfos(ctx, event)
	s.recordEvent(ctx, event)
	if s.syncHandler != nil {
		s.syncHandler.Queue(ctx, event)
	}
}

func (s *Server) recordEventInfos(ctx context.Context, event *sensor.Event) {
	if event.HardwareAddress != nil {
		s.arpCache.Put(ctx, event.Address, event.HardwareAddress)
	}
}

func (s *Server) recordEvent(ctx context.Context, event *sensor.Event) {
	// TODO: Log finalisieren
	s.logger.Info("recording event", slog.String("event", event.String()))
	serverInfo, found := s.deviceInfos.Lookup(ctx, event.Host)
	if !found {
		s.logger.Warn("failed to lookup server device info", slog.String("host", event.Host))
		return
	}
	// TODO: Log finalisieren
	s.logger.Info("server device info", slog.String("device", serverInfo.String()))
	clientInfo, found := s.deviceInfos.Lookup(ctx, event.Address.String())
	if !found {
		s.logger.Warn("failed to lookup client device info", slog.String("address", event.Address.String()))
		return
	}
	// TODO: Log finalisieren
	s.logger.Info("client device info", slog.String("device", clientInfo.String()))
	err := s.store.UpdateOrInsertConnection(ctx, serverInfo, clientInfo, event)
	if err != nil {
		s.logger.Error("failed to record connection", slog.Any("err", err))
	}
	s.metricsRecorder.RecordEvent(event, clientInfo)
}
