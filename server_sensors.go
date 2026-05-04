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
	"strings"

	"github.com/tdrn-org/go-tlsconf/tlsserver"
	"github.com/tdrn-org/netscanner/logmatcher"
	"github.com/tdrn-org/netscanner/sensor"
	"github.com/tdrn-org/netscanner/sensor/syslog"
)

func (s *Server) AddSensor(ctx context.Context, config *SensorConfig) (*sensor.Sensor, error) {
	if config.SyslogSensor != nil {
		return s.addSyslogSensor(ctx, config.SyslogSensor)
	}
	return nil, fmt.Errorf("empty sensor configuration")
}

func (s *Server) addSyslogSensor(ctx context.Context, config *SyslogSensorConfig) (*sensor.Sensor, error) {
	s.logger.Info("adding sensor", slog.Any("sensor", config))
	index, err := s.resolveLogMatcherIndex(ctx, config.LogMatcherIndex)
	if err != nil {
		return nil, err
	}
	var source syslog.Sensor
	switch config.Network {
	case "tcp", "tcp4", "tcp6":
		source, err = syslog.ListenTCP(index, string(config.Network), config.Address)
	case "tcp+tls", "tcp4+tls", "tcp6+tls":
		source, err = syslog.ListenTLS(index, strings.TrimSuffix(string(config.Network), "+tls"), config.Address, tlsserver.GetConfig())
	case "udp", "udp4", "udp6":
		source, err = syslog.ListenUDP(index, string(config.Network), config.Address)
	default:
		err = fmt.Errorf("unrecognized syslog network '%s'", config.Network)
	}
	if err != nil {
		return nil, err
	}
	sensor := sensor.New(config.Name, source)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sensors[sensor.Name()] = sensor
	return sensor, nil
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
	s.recordEventInfos(ctx, event)
	if event.Type != sensor.EventTypeInformational {
		s.recordEvent(ctx, event)
	}
}

func (s *Server) recordEventInfos(ctx context.Context, event *sensor.Event) {
	// TODO
	s.logger.Info(event.String())
}

func (s *Server) recordEvent(ctx context.Context, event *sensor.Event) {
	deviceInfo := s.deviceInfos.Lookup(ctx, event.Address)
	err := s.store.UpdateOrInsertEvent(ctx, event, deviceInfo)
	if err != nil {
		s.logger.Error("failed to record event", slog.Any("err", err))
	}
}
